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
