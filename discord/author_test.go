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

package discord

import (
	"testing"

	"github.com/ewollesen/discordgo"
	"xmtp.net/xmtpbot/test"
)

func TestNick(t *testing.T) {
	test := test.New(t)
	session := newMockSession()
	session.appendMemberNicks("foobar [dps]")
	a := &author{
		channel_id: "654321",
		session:    session,
		user: &discordgo.User{
			ID:       "123456",
			Username: "foobar",
		},
	}

	test.AssertEqual(a.Nick(), "foobar [dps]")
}

func TestNickFallsBackToUsername(t *testing.T) {
	test := test.New(t)
	session := newMockSession()
	a := &author{
		channel_id: "654321",
		session:    session,
		user: &discordgo.User{
			ID:       "123456",
			Username: "foobar",
		},
	}

	test.AssertEqual(a.Nick(), "foobar")
}
