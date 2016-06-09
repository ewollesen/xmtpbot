// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package setup

import (
	"strings"

	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/seen/json"
	"xmtp.net/xmtpbot/seen/memory"
)

func NewStore() seen.Store {
	return NewStoreFromFilename(*seen.StoreFilename)
}

func NewStoreFromFilename(filename string) seen.Store {
	switch strings.ToLower(*seen.StoreType) {
	case "json":
		return json.New(filename)
	case "memory":
	default:
	}

	return memory.New()
}
