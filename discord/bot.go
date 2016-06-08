// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package discord

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/dice"
	"xmtp.net/xmtpbot/dur"
	"xmtp.net/xmtpbot/html"
	"xmtp.net/xmtpbot/http_server"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/remind"
	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/twitch"
	"xmtp.net/xmtpbot/urls"
	"xmtp.net/xmtpbot/util"
)

var (
	token  = flag.String("discord.token", "", "Discord API token")
	logger = spacelog.GetLogger()

	DiscordError = errors.NewClass("discord")
)

type bot struct {
	handler_callbacks []func()
	seen              seen.Store
	user_id           string
	urls              urls.Store
	mildred           mildred.Conn
	remind            remind.Remind
	twitch_client     twitch.Twitch
	http_server       http_server.Server
	commands_mtx      sync.Mutex
	commands          map[string]CommandHandler
}

func New(urls_store urls.Store, seen_store seen.Store, mildred mildred.Conn,
	remind remind.Remind, twitch twitch.Twitch,
	http_server http_server.Server) *bot {

	b := &bot{
		seen:          seen_store,
		urls:          urls_store,
		mildred:       mildred,
		remind:        remind,
		twitch_client: twitch,
		http_server:   http_server,
		commands:      make(map[string]CommandHandler),
	}

	b.RegisterCommand("commands", simpleCommand(b.listCommands,
		"list available commands"))
	b.RegisterCommand("faq", staticCommand("No FAQs answered yet",
		"frequently answered questions"))
	b.RegisterCommand("help", simpleCommand(b.help,
		"list available commands"))
	b.RegisterCommand("idle", &commandHandler{
		help:    "reports a user's idle time",
		handler: b.idle,
	})
	b.RegisterCommand("link", simpleCommand(b.lookupURL,
		"search for a previously posted URL"))
	b.RegisterCommand("np", simpleCommand(b.nowPlaying,
		"report Mildred's currently playing track"))
	b.RegisterCommand("ping", staticCommand("pong", "pong"))
	b.RegisterCommand("remind", &commandHandler{
		help: "sets a reminder. " +
			"Example !remind 5 minutes take out the trash",
		handler: b.setReminder,
	})
	b.RegisterCommand("roll", simpleCommand(dice.Roll, "roll some dice"))
	b.RegisterCommand("seen", &commandHandler{
		help:    "reports when a user was last seen",
		handler: b.lastSeen,
	})
	b.RegisterCommand("syn", staticCommand("ack", "ack"))
	b.RegisterCommand("twitch", simpleCommand(b.twitch,
		"interact with twitch. Run \"!twitch help\" for more info"))
	b.RegisterCommand("url", simpleCommand(b.lookupURL,
		"search for a previously posted URL"))

	return b
}

func (b *bot) Run(shutdown chan bool, wg *sync.WaitGroup) (err error) {
	session, err := b.logIn(shutdown)
	if err != nil {
		return err
	}
	wg.Add(1)
	logger.Info("online")

	err = b.http_server.GiveRouter("twitch", b.twitch_client.ReceiveRouter)
	if err != nil {
		logger.Errore(err)
	}

	go b.http_server.Serve()

	go func() {
		<-shutdown
		logger.Infof("shutting down")
		b.logOut(session)
		wg.Done()
	}()

	return nil
}

func (b *bot) logIn(shutdown chan bool) (session *discordgo.Session, err error) {
	session, err = discordgo.New(getToken())
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	err = b.addHandlers(session, shutdown)
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	err = session.Open()
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	return session, nil
}

func getToken() (discord_token string) {
	defer func() {
		if discord_token == "" {
			logger.Warnf("discord token is blank, connection will " +
				"silenty fail")
		}
	}()

	if *token == "" {
		return os.Getenv("DISCORD_TOKEN")
	}

	return *token
}

func (b *bot) logOut(session *discordgo.Session) {
	logger.Errore(session.Close())
	logger.Errore(b.removeHandlers())
	logger.Info("offline")
}

func (b *bot) removeHandlers() (err error) {
	for _, cb := range b.handler_callbacks {
		cb()
	}

	return nil
}

func (b *bot) addHandlers(session *discordgo.Session, shutdown chan bool) (
	err error) {

	b.handler_callbacks = append(b.handler_callbacks,
		session.AddHandler(b.messageHandler))
	b.handler_callbacks = append(b.handler_callbacks,
		session.AddHandler(b.presenceHandler))

	return nil
}

func (b *bot) handleCommand(cmd Command) string {
	handler, ok := b.commands[cmd.cmd]
	if ok {
		handler.Handle(cmd)
		return "" // TODO fixme ugly
	}

	return fmt.Sprintf("unhandled command: %q", cmd.cmd)
}

type Command struct {
	cmd     string
	args    string
	session *discordgo.Session
	message *discordgo.Message
}

func (c *Command) Reply(msg string) (err error) {
	_, err = c.session.ChannelMessageSend(c.message.ChannelID, msg)
	return err
}

func (b *bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger.Debugf("<- %s", m.Content)
	logger.Warne(b.markSeen(m.Author.Username))

	if m.Author.ID == b.myDiscordUserId(s) {
		return
	}

	response := ""

	if strings.HasPrefix(m.Content, "!") {
		args := strings.SplitN(m.Content, " ", 2)
		new_args := ""
		if len(args) > 1 {
			new_args = args[1]
		}
		response = b.handleCommand(Command{
			cmd:     args[0][1:],
			args:    new_args,
			session: s,
			message: m.Message,
		})
	}

	urls := b.parseURLs(m.Content)
	if len(urls) > 0 {
		b.rememberURLs(urls...)
	}

	if response != "" {
		for _, chunk := range splitResponse(response, 10) {
			_, err := s.ChannelMessageSend(m.ChannelID, chunk)
			if err != nil {
				logger.Warne(err)
			}
		}
	}

}

func splitResponse(response string, num_lines int) (chunks []string) {
	lines := strings.Split(response, "\n")
	chunk := []string{lines[0]}
	for i := 1; i < len(lines); i++ {
		if len(chunk) >= num_lines {
			chunks = append(chunks, strings.Join(chunk, "\n"))
			chunk = []string{}
		}
		chunk = append(chunk, lines[i])
	}

	if len(chunk) > 0 {
		chunks = append(chunks, strings.Join(chunk, "\n"))
	}

	return chunks
}

func (b *bot) myDiscordUserId(s *discordgo.Session) string {
	if b.user_id != "" {
		return b.user_id
	}

	user, err := s.User("@me")
	if err != nil {
		logger.Warnf("unable to find my user id: %v", err)
		return ""
	}

	b.user_id = user.ID

	return user.ID
}

func (b *bot) presenceHandler(s *discordgo.Session, p *discordgo.PresenceUpdate) {
	logger.Warne(b.markSeen(p.User.Username))
}

func (b *bot) markSeen(name string) error {
	return b.seen.MarkSeen(name, nil)
}

func (b *bot) parseURLs(msg string) []string {
	return urls.Parse(msg)
}

func (b *bot) rememberURLs(urls ...string) {
	for _, url := range urls {
		title, err := html.ParseTitleFromURL(url)
		if err != nil {
			logger.Warne(err)
		}
		b.urls.Remember(url, title)
		logger.Debugf("remembered URL %q", url)
	}
}

func (b *bot) lookupURL(msg string) string {
	urls := b.urls.Lookup(msg)
	if len(urls) > 0 {
		lines := []string{}
		for _, url := range urls {
			lines = append(lines,
				fmt.Sprintf("%s - %s", url[0], url[1]))
		}
		sort.Strings(lines)
		return fmt.Sprintf("Matched %d URLs:\n%s",
			len(urls), strings.Join(lines, "\n"))
	}
	return "No matching URLs found"
}

func (b *bot) nowPlaying(args string) string {
	cs := b.mildred.CurrentSong()
	if cs != nil {
		return cs.String()
	} else {
		return "error determining current song"
	}
}

func (b *bot) idle(cmd Command) error {
	if cmd.args == "" {
		return cmd.Reply("No name specified")
	}

	since, err := b.seen.Idle(cmd.args)
	if err != nil {
		return cmd.Reply(fmt.Sprintf("Error retrieving idle for %q",
			cmd.args))
	}
	if since == nil {
		return cmd.Reply(fmt.Sprintf("No idle record for %q found",
			cmd.args))
	}

	return cmd.Reply(fmt.Sprintf("%s idle for %s", cmd.args, since))
}

func (b *bot) lastSeen(cmd Command) error {
	if cmd.args == "" {
		return cmd.Reply("No name specified")
	}

	at, err := b.seen.LastSeen(cmd.args)
	if err != nil {
		return cmd.Reply(fmt.Sprintf("Error retrieving last seen for %q",
			cmd.args))
	}
	if at == nil {
		return cmd.Reply(fmt.Sprintf("No seen record for %q found",
			cmd.args))
	}

	return cmd.Reply(fmt.Sprintf("%s was last seen %s", cmd.args, at))
}

func (b *bot) setReminder(cmd Command) error {
	duration, matched, err := dur.Parse(cmd.args)
	if err != nil {
		return cmd.Reply("couldn't parse reminder duration")
	}
	msg := cmd.args[len(matched):]

	b.remind.Set(time.Now().Add(*duration), func() {
		msg := fmt.Sprintf("<@%s> reminder: %s",
			cmd.message.Author.ID, msg)
		_, err := cmd.session.ChannelMessageSend(cmd.message.ChannelID,
			msg)
		logger.Warne(err)
	})

	return cmd.Reply("reminder set")
}

func (b *bot) twitch(args string) string {
	pieces := strings.SplitN(args, " ", 2)
	cmd := pieces[0]
	if len(pieces) > 1 {
		args = pieces[1]
	} else {
		args = ""
	}
	client := b.twitch_client

	switch cmd {
	case "auth":
		auth_url, err := client.Auth(args)
		if err != nil {
			return "error generating OAuth2 URL"
		}
		return fmt.Sprintf("Click here to authorize xMTP bot to access "+
			"your Twitch account: %s", auth_url)
	case "auth-follow":
		pieces := strings.Split(args, " ")
		err := client.AuthFollow(pieces[0], pieces[1:]...)
		if err != nil {
			logger.Errorf("error auth following: %v", err)
			return "error auth following"
		}
		return "Done"
	case "live":
		var response []string
		streams, err := client.Live()
		if err != nil {
			logger.Errore(err)
			return "error retrieving live twitch streams"
		}
		for _, stream := range streams {
			response = append(response,
				fmt.Sprintf("%s: %s",
					util.EscapeMarkdown(stream.Name()),
					stream.URL()))
		}
		if len(response) == 0 {
			return "no streams are live"
		}
		return strings.Join(response, "\n")
	case "follow":
		if args == "" {
			return "must specify channel name to follow"
		}
		logger.Debugf("args: %s", args)
		return client.Follow(args)
	case "unfollow":
		if args == "" {
			return "must specify channel name to unfollow"
		}
		logger.Debugf("args: %s", args)
		client.Unfollow(args)
		return "OK"
	case "following", "list":
		var response []string
		channels, err := client.Following(args)
		if err != nil {
			logger.Errore(err)
			return "error retrieving followed twitch channels"
		}
		for _, channel := range channels {
			// wrapping the link in parentheses seems to prevent the
			// discord client from expanding it?
			response = append(response,
				fmt.Sprintf("%s (%s)",
					util.EscapeMarkdown(channel.Name()), channel.URL()))
		}

		if len(channels) == 0 {
			return "no channels are followed"
		}
		return strings.Join(response, "\n")
	default:
		return fmt.Sprintf("unhandled twitch command: %q", cmd)
	}
}

type CommandHandler interface {
	Handle(Command) error
	Help() string
}

type commandHandler struct {
	help    string
	handler func(Command) error
}

func (h *commandHandler) Help() string {
	return h.help
}

func (h *commandHandler) Handle(cmd Command) (err error) {
	return h.handler(cmd)
}

func (b *bot) RegisterCommand(name string, handler CommandHandler) (err error) {
	b.commands_mtx.Lock()
	defer b.commands_mtx.Unlock()

	b.commands[name] = handler

	return nil
}

func staticCommand(response, help string) CommandHandler {
	return &commandHandler{
		help: help,
		handler: func(cmd Command) error {
			return cmd.Reply(response)
		},
	}
}

func simpleCommand(fn func(args string) string, help string) CommandHandler {
	return &commandHandler{
		help: help,
		handler: func(cmd Command) error {
			return cmd.Reply(fn(cmd.args))
		},
	}
}

func (b *bot) help(cmd string) string {
	if cmd == "" {
		return "usage: help <command>"
	}

	b.commands_mtx.Lock()
	ch, ok := b.commands[cmd]
	b.commands_mtx.Unlock()

	if !ok {
		return fmt.Sprintf("No help for %q found", cmd)
	}

	return fmt.Sprintf("%s: %s", cmd, ch.Help())
}

func (b *bot) listCommands(cmd string) string {
	b.commands_mtx.Lock()
	cmds := b.commands
	b.commands_mtx.Unlock()

	var strs []string
	for name, _ := range cmds {
		strs = append(strs, name)
	}
	sort.Strings(strs)

	return strings.Join(strs, ", ")
}
