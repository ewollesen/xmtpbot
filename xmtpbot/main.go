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

package main

import (
	"flag"
	"os"
	"os/signal"
	"path"
	"sync"

	"github.com/spacemonkeygo/flagfile"
	"github.com/spacemonkeygo/spacelog"
	spacelog_setup "github.com/spacemonkeygo/spacelog/setup"
	"xmtp.net/xmtpbot/discord"
	"xmtp.net/xmtpbot/http_server"
	"xmtp.net/xmtpbot/http_status"
	"xmtp.net/xmtpbot/mildred"
	"xmtp.net/xmtpbot/remind"
	seen_setup "xmtp.net/xmtpbot/seen/setup"
	"xmtp.net/xmtpbot/slack"
	"xmtp.net/xmtpbot/twitch"
	urls_setup "xmtp.net/xmtpbot/urls/setup"
)

var (
	configDir = flag.String("config_dir", os.ExpandEnv("$HOME/.xmtpbot"),
		"directory in which to store config and state")
	defaultFlagfile = path.Join(*configDir, "config")

	logger = spacelog.GetLoggerNamed("xmtpbot")
)

func main() {
	loadFlags()
	spacelog_setup.MustSetup("xmtpbot")

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	shutdown := make(chan bool)
	http_server := http_server.New()
	http_status := http_status.New(http_server)
	var wg sync.WaitGroup

	discord_bot := discord.New(
		urls_setup.NewStore(path.Join(*configDir, "urls.json")),
		seen_setup.NewStore(path.Join(*configDir, "seen.json")),
		mildred.New(),
		remind.New(),
		twitch.Setup(*configDir),
		http_server,
		http_status,
		nil)
	logger.Errore(discord_bot.Run(shutdown, &wg))

	slack_bot := slack.New(
		urls_setup.NewStore(path.Join(*configDir, "slack-urls.json")),
		seen_setup.NewStore(path.Join(*configDir, "slack-seen.json")),
		mildred.New(),
		http_server,
		http_status)
	logger.Errore(slack_bot.Run(shutdown, &wg))

	logger.Errore(http_status.Run(shutdown, &wg))

	go http_server.Serve()

	<-interrupt
	logger.Infof("interrupt received")
	close(shutdown)
	wg.Wait()
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
