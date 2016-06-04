// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package main

import (
	"fmt"
	"log"

	"github.com/spacemonkeygo/flagfile"
	"github.com/spacemonkeygo/spacelog"
	spacelog_setup "github.com/spacemonkeygo/spacelog/setup"

	"xmtp.net/xmtpbot/twitch"
)

var (
	logger = spacelog.GetLoggerNamed("twitch_test")
)

func main() {
	flagfile.Load()
	spacelog_setup.MustSetup("twitch_test")

	t := twitch.New()
	t.Follow("surefour", "a_seagull")

	chans, err := t.Following()
	if err != nil {
		log.Fatal(err)
	}

	for _, ch := range chans {
		fmt.Printf("here is a channel: %s: %s\n", ch.Name(), ch.URL())
	}

	streams, err := t.Live()
	if err != nil {
		log.Fatal(err)
	}

	for _, st := range streams {
		fmt.Printf("here is a stream: %s: %s\n", st.Name(), st.URL())
	}
}
