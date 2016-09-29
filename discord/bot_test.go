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

	"xmtp.net/xmtpbot/test"
)

type botTest struct {
	*test.Test
}

func TestDPSRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertOnlyDPS("foo")
	test.AssertOnlyDPS("foo []")
	test.AssertOnlyDPS("foo [garbage]")
	test.AssertOnlyDPS("foo [dps]")
	test.AssertOnlyDPS("[dps]")
	test.AssertOnlyDPS("[DPS]")
	test.AssertOnlyDPS("[Dps]")
	test.AssertOnlyDPS("[DpS]")
	test.AssertOnlyDPS("[dam]")
	test.AssertOnlyDPS("[damage]")
	test.AssertOnlyDPS("[reaper]")
	test.AssertOnlyDPS("[junkrat]")
	test.AssertOnlyDPS("[pharah]")
	test.AssertOnlyDPS("[soldier76]")
	test.AssertOnlyDPS("[s76]")
	test.AssertOnlyDPS("zenyatta [tracer]")
	test.AssertOnlyDPS("dork [dps] [I suck]")
	test.AssertOnlyDPS("tank [dps]")
	test.AssertOnlyDPS("support [dps]")
	test.AssertOnlyDPS("flex [dps]")
	test.AssertOnlyDPS("foo [supporttank]")
}

func TestSupportRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertOnlySupport("foo [support]")
	test.AssertOnlySupport("[support]")
	test.AssertOnlySupport("[supp]")
	test.AssertOnlySupport("[SUPPORT]")
	test.AssertOnlySupport("[Support]")
	test.AssertOnlySupport("[ana]")
	test.AssertOnlySupport("[lucio]")
	test.AssertOnlySupport("[mercy]")
	test.AssertOnlySupport("[zenyatta]")
	test.AssertOnlySupport("[zen]")
	test.AssertOnlySupport("dork [support] [I suck]")
	test.AssertOnlySupport("tank [support]")
	test.AssertOnlySupport("dps [support]")
	test.AssertOnlySupport("flex [support]")
}

func TestTankRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertOnlyTank("foo [tank]")
	test.AssertOnlyTank("[tank]")
	test.AssertOnlyTank("[TANK]")
	test.AssertOnlyTank("[Tank]")
	test.AssertOnlyTank("[D.Va]")
	test.AssertOnlyTank("[DVa]")
	test.AssertOnlyTank("[dva]")
	test.AssertOnlyTank("[reinhardt]")
	test.AssertOnlyTank("[rein]")
	test.AssertOnlyTank("[roadhog]")
	test.AssertOnlyTank("[road]")
	test.AssertOnlyTank("[hog]")
	test.AssertOnlyTank("[wins]")
	test.AssertOnlyTank("[winston]")
	test.AssertOnlyTank("[zarya]")
	test.AssertOnlyTank("dork [tank] [I suck]")
	test.AssertOnlyTank("support [tank]")
	test.AssertOnlyTank("dps [tank]")
	test.AssertOnlyTank("flex [tank]")
}

func TestRoleRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertRoles("foo (support)", &roles{Support: true})
	test.AssertRoles("foo :(", &roles{DPS: true})
	test.AssertRoles("foo :)", &roles{DPS: true})
	test.AssertRoles("foo [tank", &roles{DPS: true})
	test.AssertRoles("foo [mom]", &roles{DPS: true})
}

func TestSupportTankRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertRoles("foo [support/tank]", &roles{Support: true, Tank: true})
}

func TestDPSTankRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertRoles("foo [dps/tank]", &roles{DPS: true, Tank: true})
	test.AssertRoles("foo [dps tank]", &roles{DPS: true, Tank: true})
	test.AssertRoles("foo [dps,tank]", &roles{DPS: true, Tank: true})
}

func TestFlexRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertFlex("foo [flex]")
	test.AssertFlex("foo [fill]")
	test.AssertFlex("foo [any]")
	test.AssertFlex("🌺 vissy (flex) 🌺")
}

func TestDPSSupportRegExp(t *testing.T) {
	test := newBotTest(t)

	test.AssertRoles("foo [dps/support]", &roles{DPS: true, Support: true})
	test.AssertRoles("foo [dps support]", &roles{DPS: true, Support: true})
	test.AssertRoles("foo [dps,support]", &roles{DPS: true, Support: true})
}

func (t *botTest) AssertFlex(nick string, msg ...string) {
	t.AssertRoles(nick, &roles{DPS: true, Support: true, Tank: true})
}

func (t *botTest) AssertOnlyDPS(nick string, msg ...string) {
	t.AssertRoles(nick, &roles{DPS: true})
}

func (t *botTest) AssertOnlySupport(nick string, msg ...string) {
	t.AssertRoles(nick, &roles{Support: true})
}

func (t *botTest) AssertOnlyTank(nick string, msg ...string) {
	t.AssertRoles(nick, &roles{Tank: true})
}

func newBotTest(t *testing.T) *botTest {
	return &botTest{Test: test.New(t)}
}

func (t *botTest) AssertRoles(nick string, r *roles) {
	actual := extractRoles(nick)
	t.Assert(actual.DPS == r.DPS)
	t.Assert(actual.Support == r.Support)
	t.Assert(actual.Tank == r.Tank)
}
