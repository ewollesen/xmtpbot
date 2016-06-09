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
