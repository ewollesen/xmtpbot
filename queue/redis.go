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

	redis "gopkg.in/redis.v4"
)

type marshalerFn func([]byte) (Queueable, error)

type redisQueue struct {
	client    *redis.Client
	marshaler marshalerFn
	name      string
}

var _ Queue = (*redisQueue)(nil)

func NewRedis(name string, client *redis.Client, marshaler marshalerFn) Queue {
	return &redisQueue{
		name:      name,
		client:    client,
		marshaler: marshaler,
	}
}

func (q *redisQueue) Clear() error {
	_, err := q.client.LTrim(q.name, 1, 0).Result()
	if err != nil {
		return err
	}

	return nil
}

func (q *redisQueue) Dequeue(n int) (queueables []Queueable, err error) {
	err = q.client.Watch(func(tx *redis.Tx) error {
		strs, err := q.client.LRange(q.name, 0, int64(n-1)).Result()
		if err != nil {
			return err
		}

		for _, str := range strs {
			queueable, err := q.marshaler([]byte(str))
			if err != nil {
				return err
			}
			queueables = append(queueables, queueable)
			_, err = tx.MultiExec(func() error {
				_, err = q.client.LRem(q.name, 0, str).Result()
				return err
			})
			if err != nil {
				return err
			}
		}

		return nil
	}, q.name)
	if err != nil {
		return nil, err
	}

	return queueables, nil
}

func (q *redisQueue) Enqueue(queueable Queueable) error {
	bytes, err := json.Marshal(queueable)
	if err != nil {
		return err
	}

	return q.client.Watch(func(tx *redis.Tx) error {
		if pos := q.Position(queueable.Key()); pos >= 0 {
			return AlreadyQueuedError.New("")
		}

		_, err = tx.RPush(q.name, bytes).Result()
		if err != nil {
			return err
		}

		return nil
	})
}

func (q *redisQueue) List() ([]Queueable, error) {
	strs, err := q.client.LRange(q.name, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	var queueables []Queueable
	for _, str := range strs {
		queueable, err := q.marshaler([]byte(str))
		if err != nil {
			return nil, err
		}
		queueables = append(queueables, queueable)
	}

	return queueables, nil
}

func (q *redisQueue) Position(key string) int {
	queueables, err := q.List()
	if err != nil {
		logger.Errore(err)
		return -1
	}

	return q.position(key, queueables)
}

func (q *redisQueue) Remove(key string) (queueable Queueable, err error) {
	err = q.client.Watch(func(tx *redis.Tx) error {
		queueables, err := q.List()
		if err != nil {
			logger.Errore(err)
			return err
		}

		pos := q.position(key, queueables)
		if pos == -1 {
			return NotFoundError.New("")
		}

		queueable = queueables[pos-1] // pos is 1-indexed, ugh
		queueable_bytes, err := json.Marshal(queueable)
		if err != nil {
			return err
		}

		_, err = tx.LRem(q.name, 0, queueable_bytes).Result()
		if err != nil {
			return err
		}

		return nil
	}, q.name)
	if err != nil {
		return nil, err
	}

	return queueable, nil
}

func (q *redisQueue) Size() int {
	num, err := q.client.LLen(q.name).Result()
	if err != nil {
		return -1
	}

	return int(num)
}

func (q *redisQueue) position(key string, queueables []Queueable) int {
	for pos, queueable := range queueables {
		if key == queueable.Key() {
			return pos + 1
		}
	}

	return -1
}
