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
	"encoding/base64"
	"math/rand"
	"os"
	"regexp"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/spacemonkeygo/spacelog"
	spacelog_setup "github.com/spacemonkeygo/spacelog/setup"
)

var (
	btagRe = regexp.MustCompile("^\\pL[\\pL\\pN]{2,11}#\\d{1,7}$")

	logger = spacelog.GetLoggerNamed("util")
)

func init() {
	spacelog_setup.MustSetup("util")
}

func EscapeMarkdown(input string) string {
	input = strings.Replace(input, "_", "\\_", -1)
	input = strings.Replace(input, "*", "\\*", -1)

	return input
}

func RandomState(bytes int) (state string, err error) {
	buf := make([]byte, bytes)
	_, err = rand.Read(buf)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(buf), nil
}

func OpenBoltDB(db_path string) *bolt.DB {
	db, err := bolt.Open(db_path, 0600, nil)
	if err != nil {
		logger.Errore(err)
		os.Exit(1)
	}

	return db
}

func ParseBattleTag(s string) string {
	words := strings.Split(s, " ")
	for _, word := range words {
		if ValidBattleTag(word) {
			return word
		}
	}

	return ""
}

func ValidBattleTag(btag string) bool {
	return btagRe.MatchString(btag)
}
