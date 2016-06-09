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

package dur

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spacemonkeygo/spacelog"
)

var (
	durRe = regexp.MustCompile("(\\d+)\\s*(d|days?|w|weeks?|m|mins?|s|secs?|seconds?|h|hours?)")

	logger = spacelog.GetLogger()
)

func Parse(input string) (*time.Duration, string, error) {
	durRe.Longest()
	matches := durRe.FindAllStringSubmatch(strings.ToLower(input), 1)
	logger.Debugf("matches: %v", matches)
	for _, match := range matches {
		logger.Debugf("match: %v", match)
		magnitude, err := strconv.ParseInt(match[1], 10, 32)
		if err != nil {
			logger.Debugf("failed to parse magnitude: %v", err)
			continue
		}
		logger.Debugf("magnitude: %v", magnitude)

		mag := time.Duration(magnitude)
		var d time.Duration
		switch strings.ToLower(match[2][0:1]) {
		case "w":
			logger.Debugf("scale: week")
			d = time.Hour * 24 * 7 * mag
		case "d":
			logger.Debugf("scale: day")
			d = time.Hour * 24 * mag
		case "h":
			logger.Debugf("scale: hour")
			d = time.Hour * mag
		case "m":
			logger.Debugf("scale: minute")
			d = time.Minute * mag
		case "s":
			logger.Debugf("scale: second")
			d = time.Second * mag
		default:
			logger.Debugf("no case matched: %v", match[2][0:0])
			return nil, "", nil
		}

		return &d, match[0], nil
	}

	return nil, "", nil
}
