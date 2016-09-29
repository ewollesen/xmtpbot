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

import "github.com/ewollesen/discordgo"

type Author interface {
	BattleTag() (string, error) // do I belong here?
	Mention() string
	PermittedTo(perm int) (bool, error)
	Nick() (string, error)
}

type Command interface {
	Name() string
	Args() string
	Session() Session
	Message() *discordgo.Message
	Reply(template string, args ...interface{}) error
	Author() Author
}

type Session interface {
	ChannelMessageSend(channel_id, msg string) (*discordgo.Message, error)
	GuildIdFromChannelId(channel_id string) (string, error)
	Member(guild_id, user_id string) (*discordgo.Member, error)
	UserChannelPermissions(user_id string, channel_id string) (perms int,
		err error)
}
