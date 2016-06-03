// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package discord

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/dice"
	"xmtp.net/xmtpbot/html"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/urls"
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
}

func New(urls_store urls.Store, seen_store seen.Store, mildred mildred.Conn) *bot {

	return &bot{
		seen:    seen_store,
		urls:    urls_store,
		mildred: mildred,
	}
}

func (b *bot) Run(shutdown chan bool, wg *sync.WaitGroup) (err error) {
	session, err := b.logIn(shutdown)
	if err != nil {
		return err
	}
	wg.Add(1)
	logger.Info("online")

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

func (b *bot) handleCommand(cmd string, args ...string) string {
	arg_string := strings.Join(args, " ")

	switch cmd {
	case "dice":
		return dice.Roll(strings.Join(args, " "))
	case "faq":
		return "No FAQs answered yet"
	case "idle":
		if arg_string == "" {
			return "No name specified"
		}

		since, err := b.seen.Idle(arg_string)
		if err != nil {
			return fmt.Sprintf("error retrieving idle for %q",
				arg_string)
		}
		if since == nil {
			return fmt.Sprintf("No idle record for %q found",
				arg_string)
		}
		return fmt.Sprintf("%s idle for %s", arg_string, since)
	case "link":
		return b.lookupURL(arg_string)
	case "np":
		return b.nowPlaying()
	case "ping":
		return "pong"
	case "roll":
		return dice.Roll(arg_string)
	case "seen":
		if arg_string == "" {
			return "No name specified"
		}

		at, err := b.seen.LastSeen(arg_string)
		if err != nil {
			return fmt.Sprintf("error retrieving last seen for %q",
				arg_string)
		}
		if at == nil {
			return fmt.Sprintf("No seen record for %q found",
				arg_string)
		}
		return fmt.Sprintf("%s was last seen %s", arg_string, at)
	case "syn":
		return "ack"
	case "url":
		return b.lookupURL(arg_string)
	default:
		return fmt.Sprintf("unhandled command: %q", cmd)
	}
}

func (b *bot) messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	logger.Debugf("<- %s", m.Content)
	logger.Warne(b.markSeen(m.Author.Username))

	if m.Author.ID == b.myDiscordUserId(s) {
		return
	}

	response := ""

	if strings.HasPrefix(m.Content, "!") {
		args := strings.Split(m.Content, " ")
		cmd := args[0][1:]
		args = args[1:]
		response = b.handleCommand(cmd, args...)
	}

	urls := b.parseURLs(m.Content)
	if len(urls) > 0 {
		b.rememberURLs(urls...)
	}

	if response != "" {
		_, err := s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			logger.Warne(err)
		}
	}

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

func (b *bot) nowPlaying() string {
	cs := b.mildred.CurrentSong()
	if cs != nil {
		return cs.String()
	} else {
		return "error determining current song"
	}
}
