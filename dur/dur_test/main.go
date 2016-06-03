// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/spacemonkeygo/flagfile"
	spacelog_setup "github.com/spacemonkeygo/spacelog/setup"

	"xmtp.net/xmtpbot/dur"
)

var (
	testCases = map[string]time.Duration{
		"10 s":   time.Second * 10,
		"5mins":  time.Minute * 5,
		"5 Hour": time.Hour * 5,
		"2d":     time.Hour * 24 * 2,
		"1 week": time.Hour * 24 * 7,
	}
)

func main() {
	flagfile.Load()
	spacelog_setup.MustSetup("xmtpbot")

	for input, expected := range testCases {
		actual, _, err := dur.Parse(input)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("input: %q; expected: %s; actual: %s",
			input, expected, actual)

		if actual == nil || *actual != expected {
			fmt.Printf(" FAIL\n")
		} else {
			fmt.Printf(" SUCCESS\n")
		}
	}
}
