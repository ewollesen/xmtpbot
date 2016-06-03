// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package urls

import (
	"flag"
	"path"
	"regexp"

	"xmtp.net/xmtpbot/config"
)

var (
	StoreType = flag.String("urls.store_type", "json",
		"URL storage backend type")
	StoreFilename = flag.String("urls.store_filename",
		path.Join(*config.Dir, "urls.json"),
		"filename in which to store collected URLs")

	URLRegexp = regexp.MustCompile("https?://[^ ]+")
)

type Store interface {
	Clear()
	Iterate(cb func(url, title string))
	Length() int
	Lookup(msg string) (urls [][]string)
	Remember(url, title string) error
}

func Parse(input string) []string {
	return URLRegexp.FindAllString(input, -1)
}
