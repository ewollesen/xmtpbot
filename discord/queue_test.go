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
	"time"

	"github.com/bwmarrin/discordgo"

	"xmtp.net/xmtpbot/queue"
	seen_mem "xmtp.net/xmtpbot/seen/memory"
	"xmtp.net/xmtpbot/test"
	url_mem "xmtp.net/xmtpbot/urls/memory"
)

func TestDequeue(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "dequeue",
		args:    "",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID: "foobar",
			},
			ChannelID: "channel-foo",
		},
	}

	test.AssertNil(bot.dequeue(cmd))
	test.AssertEqual(len(session.replies), 1)
	assertContains(test, session.replies,
		"Successfully removed <@!foobar> from the scrimmages queue.")
}

func TestEnqueue(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "enqueue",
		args:    "",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID: "foobar",
			},
			ChannelID: "channel-foo",
		},
	}

	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 1)
	assertContains(test, session.replies,
		"Successfully added <@!foobar> to the scrimmages queue in position 1.")
}

func TestQueueClear(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "queue",
		args:    "clear",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID: "foobar",
			},
			ChannelID: "channel-foo",
		},
	}
	q := bot.queues.Lookup("channel-foo")
	test.AssertEqual(bot.queueClear(q, cmd), "Permission denied.")

	session.allowAll()
	test.AssertEqual(bot.queueClear(q, cmd), "Scrimmages queue cleared.")
}

func TestQueueList(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "queue",
		args:    "list",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID:       "123456",
				Username: "foobar",
			},
			ChannelID: "654321",
		},
	}
	q := bot.queues.Lookup("channel-foo")
	test.AssertEqual(bot.queueList(q, cmd), "The scrimmages queue is empty.")

	q.Enqueue(&user{cmd.message.Author})
	test.AssertEqual(bot.queueList(q, cmd),
		"The scrimmages queue contains 1 members: foobar.")
}

func TestQueueTake(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "take",
		args:    "",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID:       "123456",
				Username: "foobar",
			},
			ChannelID: "654321",
		},
	}
	q := bot.queues.Lookup("channel-foo")
	test.AssertEqual(bot.queueTake(q, cmd), "Permission denied.")

	session.allowAll()
	test.AssertEqual(bot.queueTake(q, cmd), "Took 0 members from the "+
		"scrimmages queue. 0 members remain in the queue.")

	q.Enqueue(&user{cmd.message.Author})
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 members from the "+
		"scrimmages queue: <@!123456>. 0 members remain in the queue.")

	q.Enqueue(&user{cmd.message.Author})
	q.Enqueue(&user{&discordgo.User{
		ID:       "234567",
		Username: "bazquuz",
	}})
	test.AssertEqual(bot.queueTake(q, cmd), "Took 2 members from the "+
		"scrimmages queue: <@!123456>, <@!234567>. 0 members remain in "+
		"the queue.")

	q.Enqueue(&user{&discordgo.User{
		ID:       "234567",
		Username: "bazquuz",
	}})
	q.Enqueue(&user{cmd.message.Author})
	cmd.args = "1"
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 members from the "+
		"scrimmages queue: <@!234567>. 1 members remain in "+
		"the queue.")
}

func newBot() *bot {
	return &bot{
		seen:          seen_mem.New(),
		urls:          url_mem.New(),
		mildred:       nil,
		remind:        nil,
		twitch_client: nil,
		http_server:   nil,
		commands:      make(map[string]CommandHandler),
		oauth_states:  make(map[string]string),
		last_activity: time.Now(),
		queues:        queue.NewManager(),
	}
}

type mockSession struct {
	perms   int
	replies []string
}

func newMockSession() *mockSession {
	return &mockSession{
		perms:   0,
		replies: make([]string, 0),
	}
}

func (s *mockSession) allowAll() {
	s.perms = 0xfffffffffffffff
}

func (s *mockSession) UserChannelPermissions(user_id, channel_id string) (
	perms int, err error) {
	return s.perms, nil
}

func (s *mockSession) ChannelMessageSend(channel_id, msg string) (
	*discordgo.Message, error) {
	s.replies = append(s.replies, msg)
	return nil, nil
}

func assertContains(test *test.Test, container []string, content string,
	msg ...string) {
	for _, c := range container {
		if c == content {
			return
		}
	}

	test.Logf("expected %+v to contain %q", container, content)
	test.Fail()
}
