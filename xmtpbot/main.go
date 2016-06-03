// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package main

import (
	"flag"
	"os"
	"os/signal"
	"sync"

	"github.com/spacemonkeygo/flagfile"
	"github.com/spacemonkeygo/spacelog"
	spacelog_setup "github.com/spacemonkeygo/spacelog/setup"

	"xmtp.net/xmtpbot/discord"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/remind"
	seen_setup "xmtp.net/xmtpbot/seen/setup"
	urls_setup "xmtp.net/xmtpbot/urls/setup"
)

var (
	defaultFlagfile = os.ExpandEnv("$HOME/.xmtpbot/config")

	logger = spacelog.GetLoggerNamed("xmtpbot")
)

func main() {
	loadFlags()
	spacelog_setup.MustSetup("xmtpbot")

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	shutdown := make(chan bool)
	bot := discord.New(
		urls_setup.NewStore(),
		seen_setup.NewStore(),
		mildred.New(),
		remind.New())
	var wg sync.WaitGroup
	logger.Errore(bot.Run(shutdown, &wg))

	select {
	case <-interrupt:
		logger.Infof("interrupt received")
		close(shutdown)
		wg.Wait()
	}
}

func loadFlags() {
	_, err := os.Stat(defaultFlagfile)
	if err == nil {
		ff := flag.Lookup("flagfile")
		ff.DefValue = defaultFlagfile
		ff.Value.Set(defaultFlagfile)
	}
	flagfile.Load()
}
