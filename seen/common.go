// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package seen

import (
	"flag"
	"path"
	"time"

	"xmtp.net/xmtpbot/config"
)

var (
	StoreType = flag.String("seen.store_type", "json",
		"seen storage backend type")
	StoreFilename = flag.String("seen.store_filename",
		path.Join(*config.Dir, "seen.json"),
		"filename in which to store last seen records")
)

type Store interface {
	MarkSeen(name string, at *time.Time) error
	LastSeen(name string) (at *time.Time, err error)
	Iterate(func(name string, at *time.Time))
	Length() int
	Idle(name string) (since *time.Duration, err error)
}
