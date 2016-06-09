// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package slack

import (
	"flag"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/nlopes/slack"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"

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

	b.RegisterCommand("ping", staticCommand("pong", "pong"))

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
