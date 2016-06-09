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

package html

import (
	"io"
	"net/http"

	"github.com/spacemonkeygo/spacelog"

	go_html "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	logger = spacelog.GetLogger()
)

func ParseTitle(r io.Reader) string {
	d := go_html.NewTokenizer(r)
	in_title := false
	title := ""

	for {
		token_type := d.Next()
		if token_type == go_html.ErrorToken {
			return ""
		}

		switch token_type {
		case go_html.StartTagToken:
			token := d.Token()
			if token.DataAtom == atom.Title {
				in_title = true
				logger.Debugf("token: %+v", token)
				continue
			}
		case go_html.TextToken:
			if in_title {
				token := d.Token()
				logger.Debugf("token: %+v", token)
				title += token.Data
				continue
			}
		case go_html.EndTagToken:
			if in_title {
				token := d.Token()
				logger.Debugf("token: %+v", token)
				return title
			}
		}

	}
}

func ParseTitleFromURL(url string) (title string, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return ParseTitle(resp.Body), nil
}
