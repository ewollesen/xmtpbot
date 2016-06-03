// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package memory

import (
	"strings"

	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/urls"
)

var (
	logger = spacelog.GetLogger()
)

type store map[string]string

func New() urls.Store {
	s := make(store)
	logger.Info("URL store initialized")

	return &s
}

func (s *store) Clear() {
	s2 := make(store)
	s = &s2
}

func (s *store) Iterate(cb func(url, title string)) {
	for url, title := range *s {
		cb(url, title)
	}
}

func (s *store) Length() (length int) {
	return len(*s)
}

func (s *store) Lookup(msg string) (urls [][]string) {
	msg = strings.ToLower(msg)
	for url, title := range *s {
		if strings.Contains(strings.ToLower(url), msg) ||
			strings.Contains(strings.ToLower(title), msg) {
			urls = append(urls, []string{url, title})
		}
	}

	return urls
}

func (s *store) Remember(url, title string) (err error) {
	if title == "" {
		(*s)[url] = url
	} else {
		(*s)[url] = title
	}

	return nil
}
