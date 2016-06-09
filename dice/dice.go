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

package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/spacemonkeygo/spacelog"
)

var (
	diceRe = regexp.MustCompile("([0-9]+)?d([0-9]+)(\\+[0-9]+)?")

	logger = spacelog.GetLoggerNamed("dice")
)

type spec struct {
	num_dice  int
	num_sides int
	modifier  int
	rolls     []int
}

func Roll(input string) string {
	logger.Debugf("Roll input: %q", input)
	var specs []*spec

	raw_specs := diceRe.FindAllStringSubmatch(input, -1)
	for _, raw_spec := range raw_specs {
		spec, err := parseSpec(raw_spec[1:])
		if err != nil {
			logger.Errore(err)
			return "?"
		}
		specs = append(specs, spec)
	}

	if len(specs) > 0 {
		return mapSpec2String(specs)
	} else {
		return mapSpec2String([]*spec{{num_dice: 1, num_sides: 6, modifier: 0}})
	}
}

func parseSpec(raw_spec []string) (s *spec, err error) {
	var num_dice int64
	var modifier int64

	logger.Debugf("raw_spec: %v", raw_spec)

	if raw_spec[0] == "" {
		num_dice = 1
	} else {
		num_dice, err = strconv.ParseInt(raw_spec[0], 10, 32)
		if err != nil {
			return nil, err
		}
	}

	num_sides, err := strconv.ParseInt(raw_spec[1], 10, 32)
	if err != nil {
		return nil, err
	}

	if raw_spec[2] == "" {
		modifier = int64(0)
	} else {
		modifier, err = strconv.ParseInt(raw_spec[2], 10, 32)
		if err != nil {
			return nil, err
		}
	}

	return &spec{
		num_dice:  int(num_dice),
		num_sides: int(num_sides),
		modifier:  int(modifier),
	}, nil
}

func (s *spec) realize() {
	if len(s.rolls) > 0 {
		return
	}

	for i := 0; i < s.num_dice; i++ {
		s.rolls = append(s.rolls, rand.Intn(s.num_sides)+1)
	}
}

func (s *spec) Sum() (total int) {
	s.realize()

	for _, roll := range s.rolls {
		total += roll
	}

	return total + s.modifier
}

func (s *spec) Rolls() []int {
	s.realize()

	return s.rolls
}

func (s *spec) String() string {
	s.realize()

	str := fmt.Sprintf("%dd%d", s.num_dice, s.num_sides)
	if s.modifier != 0 {
		str += fmt.Sprintf("+%d", s.modifier)
	}
	str += fmt.Sprintf(": %d", s.Sum())

	return str + fmt.Sprintf(" %v", s.rolls)
}

func roll(numRollsStr, sidesStr, mod string) (sum int, rolls []int, err error) {
	var numRolls int
	if numRollsStr == "" {
		numRolls = 1
	} else {
		numRolls64, err := strconv.ParseInt(numRollsStr, 10, 32)
		numRolls = int(numRolls64)
		if err != nil {
			return 0, nil, err
		}
	}

	sides, err := strconv.ParseInt(sidesStr, 10, 32)
	if err != nil {
		return 0, nil, err
	}

	for i := 0; i < int(numRolls); i++ {
		roll := rand.Intn(int(sides)) + 1
		sum += roll
		rolls = append(rolls, roll)
	}

	return sum, rolls, nil
}

func mapSpec2String(xs []*spec) (str string) {
	var pieces []string

	for _, x := range xs {
		pieces = append(pieces, x.String())
	}

	return strings.Join(pieces, ", ")
}
