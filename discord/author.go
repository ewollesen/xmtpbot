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
	"encoding/json"
	"fmt"

	"github.com/ewollesen/discordgo"
	"xmtp.net/xmtpbot/queue"
	"xmtp.net/xmtpbot/util"
)

type author struct {
	BattleTag_ string            `json:"battle_tag"`
	ChannelId  string            `json:"channel_id"`
	GuildId    string            `json:"guild_id"`
	Member_    *discordgo.Member `json:"member"`
	session    Session
	User       *discordgo.User `json:"user"`
}

var _ Author = (*author)(nil)
var _ queue.Queueable = (*author)(nil)

func newAuthor(discord_author *discordgo.User, session Session,
	channel_id string) *author {

	return &author{
		User:      discord_author,
		session:   session,
		ChannelId: channel_id,
	}
}

func (a *author) BattleTag() (string, error) {
	if a.BattleTag_ != "" {
		return a.BattleTag_, nil
	}

	a.BattleTag_ = util.ParseBattleTag(a.Nick())

	return a.BattleTag_, nil
}

func (a *author) Key() string {
	guild_id, err := a.guildId()
	if err != nil {
		guild_id = "unknown"
	}

	return fmt.Sprintf("%s-%s", guild_id, a.User.ID)
}

func (a *author) Mention() string {
	return fmt.Sprintf("<@!%s>", a.User.ID)
}

func (a *author) Nick() string {
	member, err := a.member()
	if err != nil || member.Nick == "" {
		logger.Warne(err)
		return a.User.Username
	}

	return member.Nick
}

func (a *author) PermittedTo(perm int) (bool, error) {
	perms, err := a.session.UserChannelPermissions(a.User.ID, a.ChannelId)
	if err != nil {
		return false, err
	}

	return perms&perm > 0, nil

}

func (a *author) SetBattleTag(btag string) error {
	a.BattleTag_ = btag

	return nil
}

func (a *author) guildId() (string, error) {
	if a.GuildId != "" {
		return a.GuildId, nil
	}

	guild_id, err := a.session.GuildIdFromChannelId(a.ChannelId)
	if err != nil {
		return "", nil
	}
	a.GuildId = guild_id

	return guild_id, nil
}

func (a *author) member() (*discordgo.Member, error) {
	if a.Member_ != nil {
		return a.Member_, nil
	}

	guild_id, err := a.guildId()
	if err != nil {
		return nil, err
	}

	member, err := a.session.Member(guild_id, a.User.ID)
	if err != nil {
		return nil, err
	}

	a.Member_ = member

	return member, nil
}

func AuthorMarshaler(data []byte) (queue.Queueable, error) {
	a := &author{}
	err := json.Unmarshal(data, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}
