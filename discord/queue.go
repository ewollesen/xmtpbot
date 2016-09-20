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

	"github.com/bwmarrin/discordgo"
	"xmtp.net/xmtpbot/queue"
)

const (
	defaultNumTaken = 12
)

func (b *bot) dequeue(cmd Command) (err error) {
	q := b.queues.Lookup(cmd.Message().ChannelID)

	err = q.Remove(cmd.Message().Author.ID)
	if err != nil && !queue.NotFoundError.Contains(err) {
		msg := fmt.Sprintf("Error removing %s from the queue: %s",
			cmd.Message().Author.Username, err)
		return cmd.Reply(msg)
	}

	return cmd.Reply(fmt.Sprintf("Successfully removed %s from the "+
		"scrimmages queue.", mention(&user{cmd.Message().Author})))
}

func (b *bot) enqueue(cmd Command) (err error) {
	mention := mention(&user{cmd.Message().Author})
	q := b.queues.Lookup(cmd.Message().ChannelID)

	err = q.Enqueue(&user{User: cmd.Message().Author})
	if err != nil {
		if queue.AlreadyQueuedError.Contains(err) {
			return cmd.Reply(fmt.Sprintf("User %s is already "+
				"queued in position %d.",
				mention, q.Position(cmd.Message().Author.ID)))
		}
		return cmd.Reply(fmt.Sprintf("Error enqueueing: %s", err))
	}

	return cmd.Reply(fmt.Sprintf("Successfully added %s to the scrimmages "+
		"queue in position %d.", mention, q.Size()))
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
	q := b.queues.Lookup(cmd.Message().ChannelID)

	switch cmd_name {
	case "", "help":
		msg = b.queueHelp(q, subcommand)
	case "clear":
		msg = b.queueClear(q, subcommand)
	case "dequeue", "remove", "del", "delete":
		msg = "Try `!dequeue` instead, this will be implemented later."
	case "enqueue", "add":
		msg = "Try `!enqueue` instead, this will be implemented later."
	case "list", "show":
		msg = b.queueList(q, subcommand)
	case "take", "pick", "grab":
		msg = b.queueTake(q, subcommand)
	default:
		msg = fmt.Sprintf("Unhandled scrimmages queue command: %q", cmd.Args())
	}

	return cmd.Reply(msg)
}

func (b *bot) queueHelp(q queue.Queue, cmd Command) string {
	return "Manipulates the scrimmages queue.\n`!dequeue` -- remove yourself from the scrimmages queue\n`!enqueue` -- add yourself to the scrimmages queue\n`!queue clear` -- clear the scrimmages queue\n`!queue list` -- list the members of the scrimmages queue\n`!queue pick <n>` -- removes the first `n` members from the scrimmages queue"
}

func (b *bot) queueClear(q queue.Queue, cmd Command) string {
	ok, err := userAuthorized(cmd)
	if err != nil {
		logger.Errore(err)
		return fmt.Sprintf("Error authorizing %s: %s",
			cmd.Message().Author.Username, err)
	}
	if !ok {
		return "Permission denied."
	}

	if err := q.Clear(); err != nil {
		return fmt.Sprintf("Error clearing the scrimmages queue: %s", err)
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
			names = append(names, u.(*user).Username)
		}
		return fmt.Sprintf("The scrimmages queue contains %d members: %s.",
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

	mentions := []string{}
	for _, queueable := range taken {
		u := queueable.(*user)
		mentions = append(mentions, mention(u))
	}
	msg := fmt.Sprintf("Took %d members from the scrimmages queue", len(taken))
	if len(taken) > 0 {
		msg += fmt.Sprintf(": %s.", strings.Join(mentions, ", "))
	} else {
		msg += "."
	}
	msg += fmt.Sprintf(" %d members remain in the queue.", q.Size())

	return msg
}

func mention(u *user) string {
	return fmt.Sprintf("<@!%s>", u.ID)
}

func username(user *discordgo.User) string {
	return user.Username
}

func userAuthorized(cmd Command) (ok bool, err error) {
	return userPermittedTo(cmd, discordgo.PermissionKickMembers)
}

func userPermittedTo(cmd Command, perm int) (bool, error) {
	perms, err := cmd.Session().UserChannelPermissions(
		cmd.Message().Author.ID, cmd.Message().ChannelID)
	if err != nil {
		return false, err
	}

	return perms&perm > 0, nil
}

type user struct {
	*discordgo.User
}

func (u *user) Key() string {
	return u.ID
}
