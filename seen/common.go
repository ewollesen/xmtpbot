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
