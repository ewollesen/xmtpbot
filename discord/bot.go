// Copyright 2016 Eric Wollesen <ericw at xmtp dot net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discord

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ewollesen/discordgo"
	"github.com/gorilla/mux"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"
	"xmtp.net/xmtpbot/dice"
	"xmtp.net/xmtpbot/dur"
	"xmtp.net/xmtpbot/fortune"
	"xmtp.net/xmtpbot/html"
	"xmtp.net/xmtpbot/http_server"
	"xmtp.net/xmtpbot/http_status"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/queue"
	"xmtp.net/xmtpbot/remind"
	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/twitch"
	"xmtp.net/xmtpbot/urls"
	"xmtp.net/xmtpbot/util"
)

const (
	bucketNicks      = "discord.nicks"
	bucketRoles      = "discord.roles"
	bucketBattleTags = "discord.battleTags"
)

var (
	token    = flag.String("discord.token", "", "Discord bot API token")
	clientId = flag.String("discord.client_id", "",
		"Discord application client id")
	maxSendSize = flag.Int("discord.max_send_size", 1024,
		"Max size of a single Discord message")
	hostname = flag.String("discord.hostname", "xmtpbot.xmtp.net",
		"Hostname for discord oauth redirects")
	protocol = flag.String("discord.protocol", "https",
		"Protocol for discord oauth redirects")
	game = flag.String("discord.game", "!help", "Game being played")

	roleRe     = regexp.MustCompile(`[\[\(]([^\]\)]+)[\]\)]`)
	roleTankRe = regexp.MustCompile("(?i:\\b(tank|d\\.?va|rein(hardt)?|(roadhog|road|hog)|wins(ton)?|zarya)\\b)")
	roleDPSRe  = regexp.MustCompile("(?i:\\b(dps|dam(age)?)\\b)")
	// Intentionally not including symmetra, cuz I'm a dick
	roleSupportRe = regexp.MustCompile("(?i:\\b(supp(ort)?|heal(er|s|z)?|ana|lucio|mercy|zen(yatta)?)\\b)")
	roleFlexRe    = regexp.MustCompile("(?i:\\b(any|fill|flex)\\b)")

	logger = spacelog.GetLogger()

	DiscordError = errors.NewClass("discord")
)

type bot struct {
	handler_callbacks           []func()
	seen                        seen.Store
	user_id                     string
	urls                        urls.Store
	mildred                     mildred.Conn
	remind                      remind.Remind
	twitch_client               twitch.Twitch
	http_server                 http_server.Server
	commands_mtx                sync.Mutex
	commands                    map[string]CommandHandler
	oauth_mtx                   sync.Mutex
	oauth_states                map[string]string
	last_activity               time.Time
	commands_handled            uint64
	queues                      queue.Manager
	user_enqueue_rate_limit_mtx sync.Mutex
	user_last_enqueued          map[string]time.Time
}

func New(urls_store urls.Store, seen_store seen.Store, mildred mildred.Conn,
	remind remind.Remind, twitch twitch.Twitch,
	http_server http_server.Server, http_status http_status.Status,
	queues queue.Manager) *bot {

	b := &bot{
		seen:               seen_store,
		urls:               urls_store,
		mildred:            mildred,
		remind:             remind,
		twitch_client:      twitch,
		http_server:        http_server,
		commands:           make(map[string]CommandHandler),
		oauth_states:       make(map[string]string),
		last_activity:      time.Now(),
		queues:             queues,
		user_last_enqueued: make(map[string]time.Time),
	}

	b.RegisterCommand("commands", simpleCommand(b.listCommands,
		"list available commands"))
	b.RegisterCommand("faq", staticCommand("No FAQs answered yet",
		"frequently answered questions"))
	b.RegisterCommand("fortune", simpleCommand(b.fortune,
		"receive great fortune cookie wisdom"))
	b.RegisterCommand("help", simpleCommand(b.help,
		"list available commands"))
	b.RegisterCommand("idle", &commandHandler{
		help:    "reports a user's idle time",
		handler: b.idle,
	})
	// b.RegisterCommand("link", simpleCommand(b.lookupURL,
	// 	"search for a previously posted URL"))
	if mildred != nil {
		b.RegisterCommand("np", simpleCommand(b.nowPlaying,
			"report Mildred's currently playing track"))
	}
	b.RegisterCommand("ping", staticCommand("pong", "pong"))
	//b.RegisterCommand("remind", &commandHandler{
	//		help: "sets a reminder. " +
	//			"Example !remind 5 minutes take out the trash",
	//		handler: b.setReminder,
	//	})
	b.RegisterCommand("roll", simpleCommand(dice.Roll, "roll some dice"))
	b.RegisterCommand("seen", &commandHandler{
		help:    "reports when a user was last seen",
		handler: b.lastSeen,
	})
	b.RegisterCommand("syn", staticCommand("ack", "ack"))
	if twitch != nil {
		b.RegisterCommand("twitch", simpleCommand(b.twitch,
			"interact with twitch. Run \"!twitch help\" for "+
				"more info"))
	}
	// b.RegisterCommand("url", simpleCommand(b.lookupURL,
	// 	"search for a previously posted URL"))
	if queues != nil {
		b.RegisterCommand("dequeue", &commandHandler{
			help:    "Type: `!dequeue` to leave the scrimmages queue",
			handler: b.dequeue,
		})
		b.RegisterCommand("enqueue", &commandHandler{
			help: "Type: `!enqueue MyBattleTag#1234` to enter " +
				"the scrimmages queue",
			handler: b.enqueue,
		})
		b.RegisterCommand("queue", &commandHandler{
			help: "queue for scrimmages. Run \"!queue help\" for " +
				"more info",
			handler: b.queue,
		})
	}
	http_status.Register("discord", b.Status)

	return b
}

func (b *bot) Run(shutdown chan bool, wg *sync.WaitGroup) (err error) {
	session, err := b.logIn()
	if err != nil {
		return err
	}
	wg.Add(1)
	logger.Info("online")

	if game != nil && *game != "" {
		session.UpdateStatus(0, *game)

	}

	err = b.http_server.GiveRouter("discord", b.ReceiveRouter)
	if err != nil {
		logger.Errore(err)
	}

	if b.twitch_client != nil {
		err = b.http_server.GiveRouter("twitch",
			b.twitch_client.ReceiveRouter)
		if err != nil {
			logger.Errore(err)
		}
	}

	go func() {
		<-shutdown
		logger.Infof("shutting down")
		b.logOut(session)
		wg.Done()
	}()

	return nil
}

func (b *bot) logIn() (session *discordgo.Session, err error) {
	session, err = discordgo.New(getToken())
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	err = b.addHandlers(session)
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	err = session.Open()
	if err != nil {
		return nil, DiscordError.Wrap(err)
	}

	return session, nil
}

func getTokenBare() (discord_token string) {
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

func getToken() (discord_token string) {
	// https://github.com/bwmarrin/discordgo/pull/248#issuecomment-244598442
	return fmt.Sprintf("Bot %s", getTokenBare())
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

func (b *bot) addHandlers(session *discordgo.Session) (
	err error) {

	b.handler_callbacks = append(b.handler_callbacks,
		session.AddHandler(b.messageHandler))
	b.handler_callbacks = append(b.handler_callbacks,
		session.AddHandler(b.presenceHandler))

	return nil
}

func (b *bot) handleCommand(cmd Command) {
	b.commands_mtx.Lock()
	handler, ok := b.commands[cmd.Name()]
	b.commands_mtx.Unlock()

	if ok {
		handler.Handle(cmd)
		b.commands_handled++
	}
}

type command struct {
	name    string
	args    string
	session Session
	message *discordgo.Message
	author  Author
}

func (c *command) Name() string {
	return c.name
}

func (c *command) Args() string {
	return c.args
}

func (c *command) Message() *discordgo.Message {
	return c.message
}

func (c *command) Session() Session {
	return c.session
}

func (c *command) Author() Author {
	if c.author != nil {
		return c.author
	}

	c.author = newAuthor(c.message.Author, c.session, c.Message().ChannelID)

	return c.author
}

func (c *command) Reply(template string, args ...interface{}) (err error) {
	msg := fmt.Sprintf(template, args...)

	if len(msg) < *maxSendSize {
		_, err = c.session.ChannelMessageSend(c.message.ChannelID, msg)
		return err
	}

	for _, piece := range splitResponse(msg, 10) {
		_, err = c.session.ChannelMessageSend(c.message.ChannelID, piece)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger.Debugf("<- %s", m.Content)
	logger.Warne(b.markSeen(m.Author.Username))
	b.activity()

	if m.Author.ID == b.myDiscordUserId(s) {
		return
	}

	if strings.HasPrefix(m.Content, "!") {
		args := strings.SplitN(m.Content, " ", 2)
		new_args := ""
		if len(args) > 1 {
			new_args = args[1]
		}
		b.handleCommand(&command{
			name:    args[0][1:],
			args:    new_args,
			session: &session{Session: s},
			message: m.Message,
		})
	}

	urls := b.parseURLs(m.Content)
	if len(urls) > 0 {
		b.rememberURLs(urls...)
	}
}

// TODO split on line breaks after at most X chars, not based on lines
// TODO find out the maximum message size in discord
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
	if cmd.Args() == "" {
		return cmd.Reply("No name specified")
	}

	since, err := b.seen.Idle(cmd.Args())
	if err != nil {
		return cmd.Reply("Error retrieving idle for %q", cmd.Args())
	}
	if since == nil {
		return cmd.Reply("No idle record for %q found", cmd.Args())
	}

	return cmd.Reply("%s idle for %s", cmd.Args(), since)
}

func (b *bot) lastSeen(cmd Command) error {
	if cmd.Args() == "" {
		return cmd.Reply("No name specified")
	}

	at, err := b.seen.LastSeen(cmd.Args())
	if err != nil {
		return cmd.Reply("Error retrieving last seen for %q", cmd.Args())
	}
	if at == nil {
		return cmd.Reply("No seen record for %q found", cmd.Args())
	}

	return cmd.Reply("%s was last seen %s", cmd.Args(), at)
}

func (b *bot) setReminder(cmd Command) error {
	duration, matched, err := dur.Parse(cmd.Args())
	if err != nil {
		return cmd.Reply("couldn't parse reminder duration")
	}
	msg := cmd.Args()[len(matched):]

	b.remind.Set(time.Now().Add(*duration), func() {
		msg := fmt.Sprintf("<@%s> reminder: %s",
			cmd.Message().Author.ID, msg)
		_, err := cmd.Session().ChannelMessageSend(
			cmd.Message().ChannelID, msg)
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
		streams, err := client.Live(args)
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
			return cmd.Reply(fn(cmd.Args()))
		},
	}
}

func (b *bot) help(cmd string) string {
	if cmd == "" {
		return "usage: `!help <command>`. Type `!commands` to see a " +
			"list of available commands."
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

func (b *bot) ReceiveRouter(router *mux.Router) (err error) {
	router.HandleFunc("/oauth/redirect", b.oauthRedirect)
	router.HandleFunc("/", b.handleHTTP)
	return nil
}

func (b *bot) oauthRedirect(w http.ResponseWriter, req *http.Request) {
	values := req.URL.Query()

	// The state parameter verifies that this request came from the entity
	// that we generated an OAuth request for, assuming we used HTTPs and it
	// wasn't interfered with.
	state := values.Get("state")
	if state == "" {
		http.Error(w, "failed to parse state query parameter",
			http.StatusBadRequest)
		return
	}

	b.oauth_mtx.Lock()
	_, ok := b.oauth_states[state]
	b.oauth_mtx.Unlock()
	if !ok {
		http.Error(w, "unrecognized oauth state",
			http.StatusBadRequest)
		return
	}

	// I don't believe the code is needed at all.
	code := values.Get("code")
	if code == "" {
		http.Error(w, "failed to parse code query parameter",
			http.StatusBadRequest)
		return
	}

	// Everything below here should be optional

	// Useful so that xMTP bot can greet the guild.
	guild_id := values.Get("guild_id")
	if guild_id == "" {
		logger.Warnf("no guild_id in OAuth2 redirect")
	}

	session, err := discordgo.New(getToken())
	if err != nil {
		logger.Warnf("error logging in to discord: %v", err)
	}

	logger.Debugf("guild_id: %q", guild_id)
	guild, err := session.Guild(guild_id)
	if err != nil {
		logger.Warnf("error retrieving guild information: %v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("Registered successfully"))
		return
	}

	_, err = session.ChannelMessageSend(guild_id,
		fmt.Sprintf("Hello, %s", guild.Name))
	if err != nil {
		logger.Warnf("error greeting guild: %+v", err)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("Registered successfully"))
		return
	}

	t, err := template.New("registration_successful").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8">
<title>xMTP bot Discord Integration</title>
</head>
<body>
<p>
Successfully registered xMTP bot with {{.GuildName}}.
</p>
</body>
</html>`)
	if err != nil {
		logger.Warnf("error generating success template")
		return
	}

	data := struct {
		GuildName string
	}{
		GuildName: guild.Name,
	}
	buf := bytes.NewBufferString("")
	err = t.Execute(buf, data)
	if err != nil {
		logger.Warnf("error rendering success template")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

const tpl = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>xMTP bot Discord Integration</title>
	</head>
	<body>
		<p>
		To authorize xMTP bot to join your Discord server/guild, visit the following URL: <a href="{{.URL}}">{{.URL}}</a>
		</p>
	</body>
</html>`

func (b *bot) handleHTTP(w http.ResponseWriter, req *http.Request) {
	t, err := template.New("root").Parse(tpl)
	data := struct {
		URL  string
		Test string
	}{
		URL:  b.discordOauthURL(),
		Test: url.QueryEscape("this is a test ://"),
	}
	buf := bytes.NewBufferString("")
	err = t.Execute(buf, data)
	if err != nil {
		logger.Errore(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to generate discord oauth URL"))
		return
	}
	logger.Debugf("HTTP request received\n%+v", req)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func (b *bot) discordOauthURL() string {
	params := make(url.Values)
	params.Set("response_type", "code")
	params.Set("redirect_uri",
		fmt.Sprintf("%s://%s/discord/oauth/redirect",
			*protocol, *hostname))
	params.Set("client_id", *clientId)
	params.Set("scope", "bot")
	params.Set("permission", "0")
	state, err := util.RandomState(32)
	if err != nil {
		logger.Errore(err)
		return ""
	}
	b.oauth_mtx.Lock()
	b.oauth_states[state] = time.Now().String()
	b.oauth_mtx.Unlock()
	params.Set("state", state)
	oauth_url := fmt.Sprintf("https://discordapp.com/oauth2/authorize?%s",
		params.Encode())
	logger.Debugf("discord oauth url: %s", oauth_url)

	return oauth_url
}

func (b *bot) fortune(args string) string {
	fortune, err := fortune.Fortune()
	if err != nil {
		logger.Errore(err)
		return "error retrieving fortune"
	}

	return fortune
}

func (b *bot) Status() map[string]string {
	return map[string]string{
		"urls":             fmt.Sprintf("%d", b.urls.Length()),
		"seen":             fmt.Sprintf("%d", b.seen.Length()),
		"last activity":    b.last_activity.String(),
		"commands handled": fmt.Sprintf("%d", b.commands_handled),
	}
}

func (b *bot) activity() {
	b.last_activity = time.Now()
}

type session struct {
	*discordgo.Session
}

func (s *session) Member(guild_id, user_id string) (*discordgo.Member, error) {
	return s.State.Member(guild_id, user_id)
}

func (s *session) UserChannelPermissions(user_id string, channel_id string) (
	perms int, err error) {
	return s.State.UserChannelPermissions(user_id, channel_id)
}

func (s *session) ChannelMessageSend(channel_id, msg string) (
	*discordgo.Message, error) {
	return s.Session.ChannelMessageSend(channel_id, msg)
}

func (s *session) GuildIdFromChannelId(channel_id string) (guild_id string,
	err error) {

	ch, err := s.Session.Channel(channel_id)
	if err != nil {
		return "", err
	}

	return ch.GuildID, nil
}

type roles struct {
	DPS     bool `json:"dps"`
	Support bool `json:"support"`
	Tank    bool `json:"tank"`
}

func (r *roles) hasRole() bool {
	return r.Tank || r.DPS || r.Support
}

func extractRoles(nick string) *roles {
	matches := roleRe.FindAllStringSubmatch(nick, -1)
	roles := &roles{}

	for _, match := range matches {
		for _, role := range match {
			if roleTankRe.MatchString(role) {
				roles.Tank = true
			}
			if roleDPSRe.MatchString(role) {
				roles.DPS = true
			}
			if roleSupportRe.MatchString(role) {
				roles.Support = true
			}
			if roleFlexRe.MatchString(role) {
				roles.DPS = true
				roles.Support = true
				roles.Tank = true
			}
		}
	}

	if !roles.hasRole() {
		logger.Infof("defaulting role for %q to DPS", nick)
		roles.DPS = true
	}

	return roles
}
