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

package urls

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	test := newTester(t)

	test.assertOneUrl("<https://github.com/ewollesen/xmtpbot>",
		"https://github.com/ewollesen/xmtpbot")
	test.assertOneUrl("https://user:password@github.com/foo/bar",
		"https://user:password@github.com/foo/bar")
	test.assertOneUrl("ftp://user:password@github.com/foo/bar",
		"ftp://user:password@github.com/foo/bar")
	test.assertOneUrl("http://github.com/ewollesen/xmtpbot#fragment",
		"http://github.com/ewollesen/xmtpbot#fragment")
	test.assertOneUrl(`// Copyright 2016 Eric Wollesen <ericw at xmtp dot net>
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
`,
		"http://www.apache.org/licenses/LICENSE-2.0")

	test.assertUrls("http://foobar.com/ blah http://barfoo.edu",
		[]string{"http://foobar.com/", "http://barfoo.edu"})
}

type tester struct {
	*testing.T
}

func newTester(t *testing.T) *tester {
	return &tester{
		T: t,
	}
}

func (t *tester) assertUrls(text string, urls []string, msg ...string) {
	matches := Parse(text)
	t.assertUrlMatches(matches, urls, msg...)
}

func (t *tester) assertOneUrl(text, url string, msg ...string) {
	matches := Parse(text)
	t.logFailure(len(matches) == 1, "expected %q got %q", url, matches)
	t.assertUrlMatches(matches, []string{url}, msg...)
}

func (t *tester) assertUrlMatches(matches []string, urls []string, msg ...string) {
	for i, match := range matches {
		t.logFailure(match == urls[i],
			"expected %q got %q", urls[i], match)
	}
}

func (t *tester) logFailure(cond bool, template string, args ...interface{}) {
	if !cond {
		t.Logf(fmt.Sprintf(template, args...))
		t.Fail()
	}
}
