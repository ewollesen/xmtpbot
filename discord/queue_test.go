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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ewollesen/discordgo"
	"xmtp.net/xmtpbot/queue"
	seen_mem "xmtp.net/xmtpbot/seen/memory"
	"xmtp.net/xmtpbot/test"
	url_mem "xmtp.net/xmtpbot/urls/memory"
	"xmtp.net/xmtpbot/util"
)

const (
	testChannelId = "987654"
	testGuildId   = "765432"
	testUserId    = "123456"
	testUserId2   = "234567"
	testBTag      = "example#1234"
	testBTag2     = "example#2345"
)

func TestDequeue(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)
	dequeue_cmd := newTestCommand("dequeue", "", session, msg)
	enqueue_cmd := newTestCommand("enqueue", testBTag, session, msg)

	bot.enqueue(enqueue_cmd)
	test.AssertNil(bot.dequeue(dequeue_cmd))
	test.AssertEqual(len(session.replies), 2)
	test.AssertContainsString(session.replies,
		"Successfully removed "+testBTag+" from the scrimmages queue.")
}

func TestEnqueue(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)
	var cmd *command

	cmd = newTestCommand("enqueue", testBTag, session, msg)
	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 1)
	test.AssertContainsString(session.replies,
		"Successfully added "+testBTag+" to the scrimmages queue "+
			"in position 1.")

	cmd = newTestCommand("enqueue", "", session, msg)
	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 2)
	test.AssertContainsString(session.replies,
		"No BattleTag specified. Try `!enqueue example#1234`.")

	cmd = newTestCommand("enqueue", testBTag, session, msg)
	test.AssertNil(bot.enqueue(cmd))
	test.AssertEqual(len(session.replies), 3)
	test.AssertContainsString(session.replies,
		"User <@!123456> is already queued as \"example#1234\" in position 1.")
}

func TestQueueClear(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)

	cmd := newTestCommand("queue", "clear", session, msg)
	q, err := bot.lookupQueue(testChannelId, cmd.Session())
	test.AssertNil(err)
	test.AssertEqual(bot.queueClear(q, cmd), "Permission denied.")

	session.allowAll()
	test.AssertEqual(bot.queueClear(q, cmd), "Scrimmages queue cleared.")
}

func TestQueueList(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)

	cmd := newTestCommand("queue", "list", session, msg)
	q, err := bot.lookupQueue(testChannelId, cmd.Session())
	test.AssertNil(err)
	test.AssertEqual(bot.queueList(q, cmd), "The scrimmages queue is empty.")

	cmd.Author().SetBattleTag(testBTag)
	q.Enqueue(cmd.Author())
	test.AssertEqual(bot.queueList(q, cmd),
		"The scrimmages queue contains 1 BattleTags: "+testBTag+".")
}

func TestQueueTake(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)
	q, err := bot.lookupQueue(testChannelId, session)
	test.AssertNil(err)
	var cmd *command

	cmd = newTestCommand("take", "", session, msg)
	test.AssertNil(err)
	test.AssertEqual(bot.queueTake(q, cmd), "Permission denied.")

	session.allowAll()
	cmd = newTestCommand("take", "", session, msg)
	test.AssertEqual(bot.queueTake(q, cmd), "Took 0 BattleTags from the "+
		"scrimmages queue. 0 BattleTags remain in the queue.")

	q.Enqueue(newTestAuthor(testUserId, testBTag))
	cmd = newTestCommand("take", "", session, msg)
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 BattleTags from the "+
		"scrimmages queue: "+testBTag+". 0 BattleTags remain in the queue.")

	q.Enqueue(newTestAuthor(testUserId, testBTag))
	q.Enqueue(newTestAuthor(testUserId2, testBTag2))
	cmd = newTestCommand("take", "", session, msg)
	test.AssertEqual(bot.queueTake(q, cmd), "Took 2 BattleTags from the "+
		"scrimmages queue: "+testBTag+", "+testBTag2+". 0 BattleTags remain "+
		"in the queue.")

	q.Enqueue(newTestAuthor(testUserId2, testBTag2))
	q.Enqueue(newTestAuthor(testUserId, testBTag))
	cmd = newTestCommand("take", "1", session, msg)
	test.AssertEqual(bot.queueTake(q, cmd), "Took 1 BattleTags from the "+
		"scrimmages queue: "+testBTag2+". 1 BattleTags remain in "+
		"the queue.")
}

func TestEnqueueRateLimit(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)
	q, err := bot.lookupQueue(testChannelId, session)
	test.AssertNil(err)

	bot.enqueue(newTestCommand("enqueue", testBTag, session, msg))
	bot.dequeue(newTestCommand("dequeue", "", session, msg))

	test.AssertNil(bot.enqueue(newTestCommand("enqueue", testBTag, session, msg)))
	test.AssertContainsString(session.replies,
		"Successfully added "+testBTag+" to the scrimmages"+
			" queue in position 1.")
	test.AssertContainsString(session.replies,
		"You may enqueue at most once every 5 minutes, <@!123456>. "+
			"Please try again later.")

	bot.dequeue(newTestCommand("dequeue", "", session, msg))
	test.AssertEqual(q.Size(), 0)

	bot.userEnqueued(testUserId,
		time.Now().Add(-1*(time.Minute*5+time.Second)))
	test.AssertNil(bot.enqueue(newTestCommand("enqueue", testBTag, session, msg)))

	test.AssertContainsString(session.replies,
		"Successfully added "+testBTag+" to the scrimmages "+
			"queue in position 1.")

	session.allowAll()
	bot.queueClear(q, newTestCommand("clear", "", session, msg))
	session.replies = make([]string, 0)
	test.AssertNil(bot.enqueue(newTestCommand("enqueue", testBTag, session, msg)))
	test.AssertContainsString(session.replies,
		"Successfully added "+testBTag+" to the scrimmages "+
			"queue in position 1.")
}

func TestIdentifyRoleDefaultsToDPS(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)

	bot.initBoltDb(util.OpenBoltDB("test-bolt.db")) // FIXME: ugly
	defer func() {
		os.Remove("test-bolt.db")
	}()

	cmd := newTestCommand("role", "", session, msg)
	test.AssertEqual(bot.queueIdentifyRole(nil, cmd),
		fmt.Sprintf("Roles matched by \"foobar\": DPS: %s, "+
			"Support: %s, Tank: %s",
			symbolChecked, symbolSaltire, symbolSaltire))
}

func TestIdentifyRolePerformsServerLookup(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)

	cmd := newTestCommand("role", "", session, msg)
	session.appendMemberNicks("foobar [tank]")
	test.AssertEqual(bot.queueIdentifyRole(nil, cmd),
		fmt.Sprintf("Roles matched by \"foobar [tank]\": DPS: %s, "+
			"Support: %s, Tank: %s",
			symbolSaltire, symbolSaltire, symbolChecked))
}

func TestRoles(t *testing.T) {
	test, bot, session := newQueueTest(t)
	msg := newTestMessage(testUserId, testChannelId)
	q, err := bot.lookupQueue(testChannelId, session)
	test.AssertNil(err)

	cmd := newTestCommand("roles", "", session, msg)
	test.AssertEqual(bot.queueRoles(q, cmd),
		"Roles queued in the scrimmages queue:\nTanks:\nSupports:\nDPSes:")
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

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
		bolt_db:            nil,
	}
}

type mockSession struct {
	perms   int
	replies []string
	nicks   []string
}

func newMockSession() *mockSession {
	return &mockSession{
		perms:   0,
		replies: make([]string, 0),
		nicks:   make([]string, 0),
	}
}

func (s *mockSession) allowAll() {
	s.perms = 0xfffffffffffffff
}

func (s *mockSession) appendMemberNicks(nicks ...string) {
	s.nicks = append(s.nicks, nicks...)
}

func (s *mockSession) Member(guild_id, user_id string) (*discordgo.Member, error) {
	nick := ""
	if len(s.nicks) > 0 {
		nick = s.nicks[0]
		s.nicks = s.nicks[1:]
	}

	return &discordgo.Member{
		Nick: nick,
	}, nil
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

func (s *mockSession) GuildIdFromChannelId(channel_id string) (string, error) {
	return testGuildId, nil
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

func newTestCommand(name, args string, session Session,
	message *discordgo.Message) *command {

	return &command{
		name:    name,
		args:    args,
		session: session,
		message: message,
	}
}

func newTestMessage(author_id, channel_id string) *discordgo.Message {
	return &discordgo.Message{
		Author: &discordgo.User{
			ID:       testUserId,
			Username: "foobar",
		},
		ChannelID: testChannelId,
	}
}

type queueTest struct {
	*test.Test
	bot     *bot
	session *mockSession
}

func newQueueTest(t *testing.T) (*queueTest, *bot, *mockSession) {
	qt := &queueTest{
		Test:    test.New(t),
		bot:     newBot(),
		session: newMockSession(),
	}

	return qt, qt.bot, qt.session
}

func newTestAuthor(user_id, btag string) Author {
	return &author{
		user: &discordgo.User{
			ID: user_id,
		},
		btag:     btag,
		guild_id: testGuildId,
	}
}
