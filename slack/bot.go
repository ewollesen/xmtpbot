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

package slack

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/dice"
	"xmtp.net/xmtpbot/fortune"
	"xmtp.net/xmtpbot/html"
	"xmtp.net/xmtpbot/http_server"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/urls"
)

var (
	clientId = flag.String("slack.client_id", "", "Slack bot client id")
	token    = flag.String("slack.token", "", "Slack bot auth token")

	logger = spacelog.GetLogger()

	Error = errors.NewClass("slack")
)

type bot struct {
	user_id      string
	seen         seen.Store
	urls         urls.Store
	mildred      mildred.Conn
	http_server  http_server.Server
	commands_mtx sync.Mutex
	commands     map[string]CommandHandler
	oauth_mtx    sync.Mutex
	oauth_states map[string]string
}

func New(urls_store urls.Store, seen_store seen.Store, mildred mildred.Conn,
	http_server http_server.Server) *bot {
	b := &bot{
		seen:         seen_store,
		urls:         urls_store,
		mildred:      mildred,
		http_server:  http_server,
		commands:     make(map[string]CommandHandler),
		oauth_states: make(map[string]string),
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
	b.RegisterCommand("link", simpleCommand(b.lookupURL,
		"search for a previously posted URL"))
	b.RegisterCommand("np", simpleCommand(b.nowPlaying,
		"report Mildred's currently playing track"))
	b.RegisterCommand("ping", staticCommand("pong", "pong"))
	b.RegisterCommand("roll", simpleCommand(dice.Roll, "roll some dice"))
	b.RegisterCommand("seen", &commandHandler{
		help:    "reports when a user was last seen",
		handler: b.lastSeen,
	})
	b.RegisterCommand("syn", staticCommand("ack", "ack"))
	b.RegisterCommand("url", simpleCommand(b.lookupURL,
		"search for a previously posted URL"))

	return b
}

func (b *bot) Run(shutdown chan bool, wg *sync.WaitGroup) (err error) {
	wg.Add(1)
	logger.Info("online")

	err = b.http_server.GiveRouter("slack", b.ReceiveRouter)
	if err != nil {
		logger.Errore(err)
	}

	session := slack.New(getToken())
	// session.SetDebug(true)

	rtm := session.NewRTM()
	go rtm.ManageConnection()
	go func() {
		b.messageHandler(session, rtm)
	}()

	go func() {
		<-shutdown
		logger.Infof("shutting down")
		wg.Done()
	}()

	return nil
}

func (b *bot) ReceiveRouter(router *mux.Router) (err error) {
	return nil
}

func (b *bot) messageHandler(session *slack.Client, rtm *slack.RTM) {
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			//logger.Debugf("<- %+v", msg.Data)
			switch msg.Data.(type) {
			case *slack.MessageEvent:
				me := msg.Data.(*slack.MessageEvent)
				logger.Debugf("<- %+v", me.Text)
				logger.Warne(b.markSeen(session, me.User))

				if me.User == b.mySlackUserId(rtm) {
					continue
				}

				if strings.HasPrefix(me.Text, "!") {
					args := strings.SplitN(me.Text, " ", 2)
					new_args := ""
					if len(args) > 1 {
						new_args = args[1]
					}
					b.handleCommand(Command{
						cmd:     args[0][1:],
						args:    new_args,
						rtm:     rtm,
						message: me,
					})
				}

				urls := b.parseURLs(me.Text)
				if len(urls) > 0 {
					b.rememberURLs(urls...)
				}
			}
		}
	}
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

type Command struct {
	cmd     string
	args    string
	message *slack.MessageEvent
	rtm     *slack.RTM
}

func (c *Command) Reply(msg string) (err error) {
	response := slack.OutgoingMessage{
		ID:      1,
		Type:    "message",
		Channel: c.message.Channel,
		Text:    msg,
	}
	c.rtm.SendMessage(&response)

	return nil
}

func getToken() (slack_token string) {
	if *token == "" {
		return os.Getenv("SLACK_TOKEN")
	}

	return *token
}

func (b *bot) handleCommand(cmd Command) {
	handler, ok := b.commands[cmd.cmd]
	if ok {
		err := handler.Handle(cmd)
		if err != nil {
			logger.Errore(err)
		}
	}
}

// FIXME cut and paste from discord
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

// FIXME cut and paste from discord
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

// FIXME cut and paste from discord
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

// FIXME cut and paste from discord
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

// FIXME cut and paste from discord
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

func (b *bot) markSeen(session *slack.Client, name string) error {
	user, err := session.GetUserInfo(name)
	if err != nil {
		return err
	}

	return b.seen.MarkSeen(user.Name, nil)
}

// FIXME cut and paste from discord
func (b *bot) parseURLs(msg string) []string {
	return urls.Parse(msg)
}

// FIXME cut and paste from discord
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

// FIXME cut and paste from discord
func (b *bot) nowPlaying(args string) string {
	cs := b.mildred.CurrentSong()
	if cs != nil {
		return cs.String()
	} else {
		return "error determining current song"
	}
}

func (b *bot) mySlackUserId(rtm *slack.RTM) string {
	if b.user_id != "" {
		return b.user_id
	}

	info := rtm.GetInfo()
	b.user_id = info.User.ID

	return info.User.ID
}

// FIXME cut and paste from discord
func (b *bot) fortune(args string) string {
	fortune, err := fortune.Fortune()
	if err != nil {
		logger.Errore(err)
		return "error retrieving fortune"
	}

	return fortune
}
