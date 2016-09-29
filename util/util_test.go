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

package util

import (
	"testing"

	"xmtp.net/xmtpbot/test"
)

func TestValidBattleTag(t *testing.T) {
	test := test.New(t)
	test.Assert(ValidBattleTag("example#1234"))
	test.Assert(ValidBattleTag("éxample#1234"))

	test.Assert(!ValidBattleTag("example#12345678"), "too many digits")
	test.Assert(!ValidBattleTag("3example#1234"), "can't start with a digit")
	test.Assert(!ValidBattleTag("exam ple#1234"), "no spaces")
	test.Assert(!ValidBattleTag("example"), "no discriminator")
	test.Assert(!ValidBattleTag("exam ple#"), "blank discriminator")
	test.Assert(!ValidBattleTag("exam ple#ooo"), "non-digit discriminator")
	test.Assert(!ValidBattleTag("tooooooooooooolong#1234"), "too long")
}
