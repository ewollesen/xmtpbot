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
	redis "gopkg.in/redis.v4"
	"xmtp.net/xmtpbot/discord"
	"xmtp.net/xmtpbot/http_server"
	"xmtp.net/xmtpbot/http_status"
	"xmtp.net/xmtpbot/queue"
	"xmtp.net/xmtpbot/remind"
	seen_setup "xmtp.net/xmtpbot/seen/setup"
	urls_setup "xmtp.net/xmtpbot/urls/setup"
)

var (
	configDir = flag.String("config_dir", os.ExpandEnv("$HOME/.zenbot"),
		"directory in which to store config and state")
	redisAddr = flag.String("discord.redis_addr", "localhost:6379",
		"address of redis server")
	defaultFlagfile = path.Join(*configDir, "config")

	logger = spacelog.GetLoggerNamed("zenbot")
)

func main() {
	loadFlags()
	spacelog_setup.MustSetup("zenbot")

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	shutdown := make(chan bool)
	http_server := http_server.New()
	http_status := http_status.New(http_server)
	redis_client := redis.NewClient(&redis.Options{Addr: *redisAddr, DB: 2})
	queues := queue.NewRedisManager("discord.zenbot", redis_client,
		discord.AuthorMarshaler)
	var wg sync.WaitGroup

	discord_bot := discord.New(
		urls_setup.NewStore(path.Join(*configDir, "urls.json")),
		seen_setup.NewStore(path.Join(*configDir, "seen.json")),
		nil,
		remind.New(),
		nil,
		http_server,
		http_status,
		queues)
	logger.Errore(discord_bot.Run(shutdown, &wg))
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
