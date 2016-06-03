// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package setup

import (
	"strings"

	"xmtp.net/xmtpbot/urls"
	"xmtp.net/xmtpbot/urls/json"
	"xmtp.net/xmtpbot/urls/memory"
)

func NewStore() urls.Store {
	switch strings.ToLower(*urls.StoreType) {
	case "json":
		return json.New(*urls.StoreFilename)
	case "memory":
	default:
	}

	return memory.New()
}
