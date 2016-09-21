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

const (
	testChannelId = "987654"
	testUserId    = "123456"
	testUserId2   = "234567"
	testBTag      = "example#1234"
	testBTag2     = "example#2345"
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
				ID: testUserId,
			},
			ChannelID: testChannelId,
		},
	}
	enqueue_cmd := &command{
		name:    "enqueue",
		args:    testBTag,
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID: testUserId,
			},
			ChannelID: testChannelId,
		},
	}

	bot.enqueue(enqueue_cmd)
	test.AssertNil(bot.dequeue(cmd))
	test.AssertEqual(len(session.replies), 2)
	assertContains(test, session.replies,
		"Successfully removed "+testBTag+" from the scrimmages queue.")
}

func TestEnqueue(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "enqueue",
		args:    testBTag,
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID: testUserId,
			},
			ChannelID: testChannelId,
		},
	}

	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 1)
	assertContains(test, session.replies,
		"Successfully added "+testBTag+" to the scrimmages queue "+
			"in position 1.")

	cmd.args = ""
	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 2)
	assertContains(test, session.replies,
		"No BattleTag specified. Try `!enqueue example#1234`.")
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
				ID: testUserId,
			},
			ChannelID: testChannelId,
		},
	}
	q := bot.queues.Lookup(testChannelId)
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
				ID:       testUserId,
				Username: "foobar",
			},
			ChannelID: testChannelId,
		},
	}
	q := bot.queues.Lookup(testChannelId)
	test.AssertEqual(bot.queueList(q, cmd), "The scrimmages queue is empty.")

	q.Enqueue(&user{User: cmd.message.Author, btag: testBTag})
	test.AssertEqual(bot.queueList(q, cmd),
		"The scrimmages queue contains 1 BattleTags: "+testBTag+".")
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
				ID:       testUserId,
				Username: "foobar",
			},
			ChannelID: testChannelId,
		},
	}
	q := bot.queues.Lookup(testChannelId)
	test.AssertEqual(bot.queueTake(q, cmd), "Permission denied.")

	session.allowAll()
	test.AssertEqual(bot.queueTake(q, cmd), "Took 0 BattleTags from the "+
		"scrimmages queue. 0 BattleTags remain in the queue.")

	q.Enqueue(&user{User: cmd.message.Author, btag: testBTag})
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 BattleTags from the "+
		"scrimmages queue: "+testBTag+". 0 BattleTags remain in the queue.")

	q.Enqueue(&user{User: cmd.message.Author, btag: testBTag})
	q.Enqueue(&user{User: &discordgo.User{
		ID:       testUserId2,
		Username: "bazquuz",
	}, btag: testBTag2})
	test.AssertEqual(bot.queueTake(q, cmd), "Took 2 BattleTags from the "+
		"scrimmages queue: "+testBTag+", "+testBTag2+". 0 BattleTags remain "+
		"in the queue.")

	q.Enqueue(&user{User: &discordgo.User{
		ID:       testUserId2,
		Username: "bazquuz",
	}, btag: testBTag2})
	q.Enqueue(&user{User: cmd.message.Author, btag: testBTag})
	cmd.args = "1"
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 BattleTags from the "+
		"scrimmages queue: "+testBTag2+". 1 BattleTags remain in "+
		"the queue.")
}

func TestEnqueueRateLimit(t *testing.T) {
	test := test.New(t)
	bot := newBot()
	session := newMockSession()
	cmd := &command{
		name:    "enqueue",
		args:    testBTag,
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID:       testUserId,
				Username: "foobar",
			},
			ChannelID: testChannelId,
		},
	}

	test.AssertNil(bot.enqueue(cmd))
	test.AssertNil(bot.enqueue(cmd))
	assertContains(test, session.replies,
		"Successfully added "+testBTag+" to the scrimmages"+
			" queue in position 1.")
	assertContains(test, session.replies,
		"You may enqueue at most once every 5 minutes, <@!123456>. "+
			"Please try again later.")

	dequeue_cmd := &command{
		name:    "dequeue",
		args:    "",
		session: session,
		message: &discordgo.Message{
			Author: &discordgo.User{
				ID:       testUserId,
				Username: "foobar",
			},
			ChannelID: testChannelId,
		},
	}

	bot.dequeue(dequeue_cmd)
	q := bot.queues.Lookup(testChannelId)
	test.AssertEqual(q.Size(), 0)

	bot.userEnqueued(testUserId,
		time.Now().Add(-1*(time.Minute*5+time.Second)))
	test.AssertNil(bot.enqueue(cmd))
	assertContains(test, session.replies,
		"Successfully added "+testBTag+" to the scrimmages "+
			"queue in position 1.")

	session.allowAll()
	bot.queueClear(q, cmd)
	session.replies = make([]string, 0)
	test.AssertNil(bot.enqueue(cmd))
	assertContains(test, session.replies,
		"Successfully added "+testBTag+" to the scrimmages "+
			"queue in position 1.")

}

func TestValidBattleTag(t *testing.T) {
	test := test.New(t)
	test.Assert(validBattleTag("example#1234"))
	test.Assert(validBattleTag("éxample#1234"))

	test.Assert(!validBattleTag("example#12345678"), "too many digits")
	test.Assert(!validBattleTag("3example#1234"), "can't start with a digit")
	test.Assert(!validBattleTag("exam ple#1234"), "no spaces")
	test.Assert(!validBattleTag("example"), "no discriminator")
	test.Assert(!validBattleTag("exam ple#"), "blank discriminator")
	test.Assert(!validBattleTag("exam ple#ooo"), "non-digit discriminator")
	test.Assert(!validBattleTag("tooooooooooooolong#1234"), "too long")
}

func newBot() *bot {
	return &bot{
		seen:               seen_mem.New(),
		urls:               url_mem.New(),
		mildred:            nil,
		remind:             nil,
		twitch_client:      nil,
		http_server:        nil,
		commands:           make(map[string]CommandHandler),
		oauth_states:       make(map[string]string),
		last_activity:      time.Now(),
		queues:             queue.NewManager(),
		user_last_enqueued: make(map[string]time.Time),
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
