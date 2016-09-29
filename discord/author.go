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

	"github.com/ewollesen/discordgo"
	"xmtp.net/xmtpbot/queue"
	"xmtp.net/xmtpbot/util"
)

type author struct {
	user       *discordgo.User
	session    Session
	channel_id string
	guild_id   string
	btag       string
}

var _ Author = (*author)(nil)
var _ queue.Queueable = (*author)(nil)

func (a *author) BattleTag() (string, error) {
	if a.btag != "" {
		return a.btag, nil
	}

	nick, err := a.Nick()
	if err != nil {
		return "", nil
	}

	a.btag = util.ParseBattleTag(nick)

	return a.btag, nil
}

func (a *author) Key() string {
	guild_id, err := a.guildId()
	if err != nil {
		guild_id = "unknown"
	}

	return fmt.Sprintf("%s-%s", guild_id, a.user.ID)
}

func (a *author) Mention() string {
	return fmt.Sprintf("<@!%s>", a.user.ID)
}

func (a *author) Nick() (string, error) {
	guild_id, err := a.guildId()
	if err != nil {
		return "", err
	}

	member, err := a.session.Member(guild_id, a.user.ID)
	if err != nil {
		return "", err
	}

	if member.Nick != "" {
		return member.Nick, nil
	}

	return a.user.Username, nil
}

func (a *author) PermittedTo(perm int) (bool, error) {
	perms, err := a.session.UserChannelPermissions(a.user.ID, a.channel_id)
	if err != nil {
		return false, err
	}

	return perms&perm > 0, nil

}

func (a *author) guildId() (string, error) {
	if a.guild_id != "" {
		return a.guild_id, nil
	}

	guild_id, err := a.session.GuildIdFromChannelId(a.channel_id)
	if err != nil {
		return "", nil
	}
	a.guild_id = guild_id

	return guild_id, nil
}
