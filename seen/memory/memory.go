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

package memory

import (
	"strings"
	"time"

	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/seen"
)

var (
	logger = spacelog.GetLogger()
)

type store map[string]*time.Time

func New() seen.Store {
	s := make(store)
	logger.Info("seen store initialized")

	return &s
}

func (ss *store) MarkSeen(name string, at *time.Time) (err error) {
	if at == nil {
		t := time.Now()
		(*ss)[strings.ToLower(name)] = &t
		return nil
	}

	(*ss)[name] = at

	return nil
}

func (ss *store) LastSeen(name string) (at *time.Time, err error) {
	at, ok := (*ss)[strings.ToLower(name)]
	if !ok {
		return nil, nil
	}

	return at, nil
}

func (ss *store) Iterate(f func(name string, at *time.Time)) {
	for n, a := range *ss {
		f(n, a)
	}
}

func (ss *store) Length() int {
	return len(*ss)
}

func (ss *store) Idle(name string) (since *time.Duration, err error) {
	at, err := ss.LastSeen(name)
	if err != nil {
		return nil, err
	}

	if at == nil {
		return nil, nil
	}

	s := time.Since(*at)

	return &s, nil
}
