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
	"strconv"
	"strings"
	"time"

	"github.com/ewollesen/discordgo"
	"xmtp.net/xmtpbot/queue"
	"xmtp.net/xmtpbot/util"
)

const (
	defaultNumTaken = 12
)

var (
	symbolChecked = string([]byte{0xe2, 0x9c, 0x93})
	symbolSaltire = string([]byte{0xe2, 0x98, 0x93})
)

func (b *bot) dequeue(cmd Command) (err error) {
	q, err := b.lookupQueue(cmd.Message().ChannelID, cmd.Session())
	if err != nil {
		logger.Errore(err)
		return cmd.Reply("Error looking up guild: %s", err)
	}

	queueable, err := q.Remove(cmd.Author().Key())
	if err != nil && !queue.NotFoundError.Contains(err) {
		return cmd.Reply("Error removing %s from the queue: %s",
			cmd.Author().Nick(), err)
	}

	if queueable == nil {
		logger.Error("Removed nil member from the queue")
		return cmd.Reply("Error removing %s from the queue.",
			cmd.Author().Nick())
	}

	a := queueable.(Author)
	btag, err := a.BattleTag()
	if err != nil {
		btag = a.Nick()
	}

	return cmd.Reply("Successfully removed %s from the scrimmages "+
		"queue.", btag)
}

func (b *bot) userEnqueueRateLimitTriggered(key string) bool {
	b.user_enqueue_rate_limit_mtx.Lock()
	defer b.user_enqueue_rate_limit_mtx.Unlock()

	at, ok := b.user_last_enqueued[key]
	if !ok {
		return false
	}

	return at.Add(time.Minute * 5).After(time.Now())
}

func (b *bot) userEnqueued(key string, at time.Time) {
	b.user_enqueue_rate_limit_mtx.Lock()
	defer b.user_enqueue_rate_limit_mtx.Unlock()

	b.user_last_enqueued[key] = at
}

func (b *bot) enqueue(cmd Command) (err error) {
	btag := ""
	args := strings.Split(cmd.Args(), " ")
	if len(args) > 0 && args[0] != "" {
		btag = args[0]
	}

	if btag == "" {
		btag, err := cmd.Author().BattleTag()
		if err != nil || btag == "" {
			return cmd.Reply("No BattleTag specified. " +
				"Try `!enqueue example#1234`.")
		}
	}
	if !util.ValidBattleTag(btag) {
		return cmd.Reply(
			fmt.Sprintf("BattleTag %q appears to be invalid.", btag))
	}

	cmd.Author().SetBattleTag(btag)

	q, err := b.lookupQueue(cmd.Message().ChannelID, cmd.Session())
	if err != nil {
		logger.Errore(err)
		return cmd.Reply("Error looking up guild: %s", err)
	}

	pos := q.Position(cmd.Author().Key())
	if pos > -1 {
		return cmd.Reply("User %s is already queued as %q "+
			"in position %d.", cmd.Author().Mention(), btag, pos)
	}

	if b.userEnqueueRateLimitTriggered(cmd.Author().Key()) {
		return cmd.Reply("You may enqueue at most once every 5 "+
			"minutes, %s. Please try again later.",
			cmd.Author().Mention())
	}

	err = q.Enqueue(cmd.Author())
	if err != nil {
		if queue.AlreadyQueuedError.Contains(err) {
			return cmd.Reply("User %s is already "+
				"queued as %q in position %d.",
				cmd.Author().Mention(), btag,
				q.Position(cmd.Author().Key()))
		}
		return cmd.Reply("Error enqueueing: %s", err)
	}

	b.userEnqueued(cmd.Author().Key(), time.Now())

	return cmd.Reply("Successfully added %s to the scrimmages "+
		"queue in position %d.", btag, q.Size())
}

func (b *bot) queue(cmd Command) (err error) {
	msg := ""
	pieces := strings.SplitN(cmd.Args(), " ", 3)
	cmd_name := pieces[0]
	subcommand := &command{
		message: cmd.Message(),
		session: cmd.Session(),
	}
	if len(pieces) > 1 {
		subcommand.name = pieces[1]
	}
	if len(pieces) > 2 {
		subcommand.args = pieces[2]
	}
	q, err := b.lookupQueue(cmd.Message().ChannelID, cmd.Session())
	if err != nil {
		logger.Errore(err)
		return cmd.Reply("Error looking up guild: %s", err)
	}

	switch cmd_name {
	case "", "help":
		msg = b.queueHelp(q, subcommand)
	case "clear":
		msg = b.queueClear(q, subcommand)
	case "dequeue", "remove", "del", "delete":
		msg = "Try `!dequeue` instead, this will be implemented later."
	case "enqueue", "add":
		msg = "Try `!enqueue` instead, this will be implemented later."
	case "id", "identify":
		msg = b.queueIdentifyRole(q, subcommand)
	case "list", "show":
		msg = b.queueList(q, subcommand)
	case "take", "pick", "grab":
		msg = b.queueTake(q, subcommand)
	case "role", "roles":
		msg = b.queueRoles(q, subcommand)
	default:
		msg = fmt.Sprintf("Unhandled scrimmages queue command: %q", cmd.Args())
	}

	return cmd.Reply(msg)
}

func (b *bot) queueHelp(q queue.Queue, cmd Command) string {
	return "Manipulates the scrimmages queue.\n`!dequeue` -- remove yourself from the scrimmages queue\n`!enqueue MyBattleTag#1234` -- add your BattleTag to the scrimmages queue\n`!queue clear` -- clear the scrimmages queue\n`!queue list` -- list the BattleTags of the scrimmages queue\n`!queue pick <n>` -- removes the first `n` BattleTags from the scrimmages queue\n`!queue identify` -- display the roles that your nickname matches (ie DPS, support, or tank)"
}

func (b *bot) queueClear(q queue.Queue, cmd Command) string {
	ok, err := userAuthorized(cmd)
	if err != nil {
		logger.Errore(err)
		return fmt.Sprintf("Error authorizing %s: %s",
			cmd.Author().Nick(), err)
	}
	if !ok {
		return "Permission denied."
	}

	if err := q.Clear(); err != nil {
		return fmt.Sprintf("Error clearing the scrimmages queue: %s", err)
	}
	if err = b.clearUserLastEnqueued(); err != nil {
		logger.Errore(err)
	}

	return "Scrimmages queue cleared."
}

func (b *bot) queueList(q queue.Queue, cmd Command) string {
	users, err := q.List()
	if err != nil {
		logger.Errore(err)
		return fmt.Sprintf("Error listing the scrimmages queue: %s", err)
	}

	if len(users) > 0 {
		names := []string{}
		for _, u := range users {
			names = append(names, u.(*user).btag)
		}
		return fmt.Sprintf("The scrimmages queue contains %d BattleTags: %s.",
			len(names), strings.Join(names, ", "))
	} else {
		return "The scrimmages queue is empty."
	}
}

func (b *bot) queueTake(q queue.Queue, cmd Command) string {
	ok, err := userAuthorized(cmd)
	if err != nil {
		logger.Errore(err)
		return fmt.Sprintf("Error authorizing %s: %s",
			cmd.Message().Author.Username, err)
	}
	if !ok {
		return "Permission denied."
	}

	num := int64(defaultNumTaken)
	args := strings.Split(cmd.Args(), " ")
	if len(args) > 0 && args[0] != "" {
		num, err = strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return fmt.Sprintf("Invalid take argument %q.", args[0])
		}
	}
	taken, err := q.Dequeue(int(num))
	if err != nil {
		logger.Errore(err)
		return fmt.Sprintf("Error taking %d members from the "+
			"scrimmages queues: %s", num, err)
	}

	btags := []string{}
	for _, queueable := range taken {
		u := queueable.(*user)
		btags = append(btags, u.btag)
	}
	msg := fmt.Sprintf("Took %d BattleTags from the scrimmages queue", len(taken))
	if len(taken) > 0 {
		msg += fmt.Sprintf(": %s.", strings.Join(btags, ", "))
	} else {
		msg += "."
	}
	msg += fmt.Sprintf(" %d BattleTags remain in the queue.", q.Size())

	return msg
}

func (b *bot) queueIdentifyRole(q queue.Queue, cmd Command) string {
	nick := cmd.Author().Nick()
	roles := extractRoles(nick)
	dps := symbolSaltire
	if roles.DPS {
		dps = symbolChecked
	}
	support := symbolSaltire
	if roles.Support {
		support = symbolChecked
	}
	tank := symbolSaltire
	if roles.Tank {
		tank = symbolChecked
	}
	msg := fmt.Sprintf("Roles matched by %q: DPS: %s, Support: %s, Tank: %s",
		nick, dps, support, tank)

	return msg
}

func (b *bot) queueRoles(q queue.Queue, cmd Command) string {
	tanks := "Tanks:"
	supports := "Supports:"
	dps := "DPSes:"

	return "Roles queued in the scrimmages queue:\n" +
		strings.Join([]string{tanks, supports, dps}, "\n")
}

func userAuthorized(cmd Command) (ok bool, err error) {
	return cmd.Author().PermittedTo(discordgo.PermissionKickMembers)
}

type user struct {
	*discordgo.User
	btag string
}

func (u *user) Key() string {
	return u.ID
}

func (b *bot) clearUserLastEnqueued() (err error) {
	b.user_enqueue_rate_limit_mtx.Lock()
	defer b.user_enqueue_rate_limit_mtx.Unlock()

	b.user_last_enqueued = make(map[string]time.Time)

	return nil
}

func (b *bot) lookupQueue(channel_id string, session Session) (queue.Queue,
	error) {

	guild_id, err := session.GuildIdFromChannelId(channel_id)
	if err != nil {
		return nil, err
	}

	return b.queues.Lookup(guild_id), nil
}
