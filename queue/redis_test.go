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
	"encoding/json"
	"testing"

	redis "gopkg.in/redis.v4"
	"xmtp.net/xmtpbot/test"
)

func TestRedisClear(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("bar", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("quux", "bar")))
	test.AssertEqual(4, test.queue.Size())
	test.AssertNil(test.queue.Clear())
	test.AssertEqual(0, test.queue.Size())
}

func TestRedisDequeue(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	items, err := test.queue.Dequeue(1)
	test.AssertNil(err)
	test.AssertEqual(0, test.queue.Size())
	test.AssertEqual(1, len(items))
	test.AssertEqual("foo", items[0].Key())
}

func TestRedisEnqueue(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertEqual(1, test.queue.Size())
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "quux")))
	test.AssertEqual(2, test.queue.Size())

	test.AssertErrorContains(test.queue.Enqueue(newQueueable("foo", "bar")),
		AlreadyQueuedError)
}

func TestRedisList(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "quux")))
	items, err := test.queue.List()
	test.AssertNil(err)
	test.AssertEqual(2, len(items))
	test.AssertEqual("foo", items[0].Key())
	test.AssertEqual("baz", items[1].Key())
}

func TestRedisPosition(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "quux")))

	test.AssertEqual(-1, test.queue.Position("deadbeef"))
	test.AssertEqual(1, test.queue.Position("foo"))
	test.AssertEqual(2, test.queue.Position("baz"))
}

func TestRedisRemove(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "quux")))
	test.AssertNil(test.queue.Enqueue(newQueueable("quux", "foo")))
	test.AssertEqual(3, test.queue.Size())

	item, err := test.queue.Remove("baz")
	test.AssertNil(err)
	test.AssertEqual("baz", item.Key())
	test.AssertEqual(2, test.queue.Size())

	_, err = test.queue.Remove("deadbeef")
	test.AssertErrorContains(err, NotFoundError)
}

func TestRedisSize(t *testing.T) {
	test := newRedisTest(t)
	defer test.Close()

	test.AssertEqual(0, test.queue.Size())
	test.AssertNil(test.queue.Enqueue(newQueueable("foo", "bar")))
	test.AssertEqual(1, test.queue.Size())
	test.AssertNil(test.queue.Enqueue(newQueueable("baz", "quux")))
	test.AssertEqual(2, test.queue.Size())
	test.AssertNil(test.queue.Enqueue(newQueueable("dead", "beef")))
	test.AssertEqual(3, test.queue.Size())

	_, err := test.queue.Dequeue(1)
	test.AssertNil(err)
	test.AssertEqual(2, test.queue.Size())
}

type redisTest struct {
	*test.Test
	queue Queue
}

func (rt *redisTest) Close() {
	rt.AssertNil(rt.queue.Clear())
}

func newRedisTest(t *testing.T) *redisTest {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	rt := &redisTest{
		queue: NewRedis("xmtpbot.testing", client,
			func(data []byte) (Queueable, error) {
				tq := &testQueueable{}
				err := json.Unmarshal(data, tq)
				if err != nil {
					return nil, err
				}

				return tq, nil
			}),
		Test: test.New(t),
	}

	rt.AssertNil(rt.queue.Clear())

	return rt
}
