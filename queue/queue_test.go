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

package queue

import (
	"testing"

	"xmtp.net/xmtpbot/test"
)

func TestEnqueue(t *testing.T) {
	test := test.New(t)
	q := New()

	q.Enqueue(newQueueable("foo", "bar"))
	test.Assert(q.Size() == 1)
	assertContains(test, q, "foo")

	test.AssertNil(q.Enqueue(newQueueable("baz", "quux")))
	test.Assert(q.Size() == 2)
	assertContains(test, q, "foo")
	assertContains(test, q, "baz")

	test.AssertErrorContains(q.Enqueue(newQueueable("foo", "blugh")),
		AlreadyQueuedError)
	test.Assert(q.Size() == 2)
}

func TestClear(t *testing.T) {
	test := test.New(t)
	q := New()
	q.Enqueue(newQueueable("foo", "bar"))
	q.Enqueue(newQueueable("baz", "quux"))

	test.Assert(q.Size() == 2)
	test.AssertNil(q.Clear())
	test.Assert(q.Size() == 0)

	test.AssertNil(q.Clear())
}

func TestList(t *testing.T) {
	test := test.New(t)
	q := New()

	users, err := q.List()
	test.AssertNil(err)
	test.Assert(len(users) == 0)

	q.Enqueue(newQueueable("foo", "bar"))
	users, err = q.List()
	test.AssertNil(err)
	test.Assert(len(users) == 1)

	q.Enqueue(newQueueable("baz", "quux"))
	users, err = q.List()
	test.AssertNil(err)
	test.Assert(len(users) == 2)
}

func TestDequeue(t *testing.T) {
	test := test.New(t)
	q := New()
	q.Enqueue(newQueueable("foo", "bar"))
	q.Enqueue(newQueueable("baz", "quux"))

	users, err := q.Dequeue(1)
	test.AssertNil(err)
	test.Assert(len(users) == 1)
	test.Assert(q.Size() == 1)
	test.Assert(users[0].Key() == "foo")

	q.Enqueue(newQueueable("foo", "bar"))
	users, err = q.Dequeue(1)
	test.AssertNil(err)
	test.Assert(len(users) == 1)
	test.Assert(q.Size() == 1)
	test.Assert(users[0].Key() == "baz")

	users, err = q.Dequeue(1)
	test.AssertNil(err)
	test.Assert(len(users) == 1)
	test.Assert(q.Size() == 0)
	test.Assert(users[0].Key() == "foo")

	users, err = q.Dequeue(1)
	test.AssertNil(err)
	test.Assert(len(users) == 0)
	test.Assert(q.Size() == 0)
}

func TestPosition(t *testing.T) {
	test := test.New(t)
	q := New()

	test.Assert(-1 == q.Position("foo"))

	q.Enqueue(newQueueable("foo", "bar"))
	test.Assert(1 == q.Position("foo"), "foo should be first")

	q.Enqueue(newQueueable("baz", "quux"))
	test.AssertEqual(q.Position("foo"), 1, "foo should be first")
	test.AssertEqual(q.Position("baz"), 2, "baz should be second")

	q.Remove("foo")
	test.AssertEqual(q.Position("foo"), -1, "foo shouldn't be found")
	test.AssertEqual(q.Position("baz"), 1, "baz should be first")
}

func TestRemove(t *testing.T) {
	test := test.New(t)
	q := New()

	test.AssertErrorContains(q.Remove("foo"), NotFoundError)

	q.Enqueue(newQueueable("foo", "bar"))
	test.AssertNil(q.Remove("foo"))

	test.AssertErrorContains(q.Remove("foo"), NotFoundError)
}

func assertContains(t *test.Test, q *queue, id string) {
	t.Assert(q.contains(id))
}

type testQueueable struct {
	key   string
	value string
}

func (q *testQueueable) Key() string {
	return q.key
}

func newQueueable(key, value string) *testQueueable {
	return &testQueueable{
		key:   key,
		value: value,
	}
}
