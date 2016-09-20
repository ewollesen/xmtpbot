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
	"sync"

	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"
)

var (
	logger = spacelog.GetLogger()

	Error              = errors.NewClass("queue")
	AlreadyQueuedError = Error.NewClass("user is already queued",
		errors.NoCaptureStack())
	NotFoundError = Error.NewClass("user is not queued",
		errors.NoCaptureStack())
)

func (p *User) Id() string {
	return p.id
}

func (p *User) Name() string {
	return p.name
}

type queue struct {
	users []User
	mtx   sync.Mutex
}

func New() Queue {
	return &queue{
		users: make([]User, 0, 100),
	}
}

func (q *queue) Add(id string, name string) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.contains(id) {
		return AlreadyQueuedError.New("")
	}
	q.users = append(q.users, User{id: id, name: name})

	return nil
}

func (q *queue) Clear() error {
	q.mtx.Lock()
	q.users = []User{}
	q.mtx.Unlock()
	return nil
}

func (q *queue) List() ([]User, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	return q.users, nil
}

func (q *queue) Pick(amount int) ([]User, error) {
	logger.Debugf("Pick amount %d", amount)
	if amount == 0 {
		return nil, nil
	}

	q.mtx.Lock()
	defer q.mtx.Unlock()

	num_picked := min(len(q.users), amount)
	picked := q.users[:num_picked]
	q.users = q.users[num_picked:]

	return picked, nil
}

func (q *queue) Remove(id string) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if !q.contains(id) {
		return NotFoundError.New("")
	}
	without := []User{}
	for _, candidate := range q.users {
		if id != candidate.id {
			without = append(without, candidate)
		}
	}
	q.users = without

	return nil
}

func (q *queue) Size() int {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	return len(q.users)
}

// You should hold q.mtx when you call this
func (q *queue) contains(id string) bool {
	for _, candidate := range q.users {
		if id == candidate.id {
			return true
		}
	}

	return false
}

func (q *queue) Position(id string) int {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for idx, candidate := range q.users {
		if candidate.id == id {
			return idx + 1
		}
	}

	return -1
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
