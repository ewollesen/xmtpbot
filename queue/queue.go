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

	"github.com/spacemonkeygo/spacelog"
)

var (
	logger = spacelog.GetLogger()
)

type queue struct {
	queueables []Queueable
	mtx        sync.Mutex
}

func New() *queue {
	return &queue{
		queueables: make([]Queueable, 0, 100),
	}
}

func (q *queue) Clear() error {
	q.mtx.Lock()
	q.queueables = []Queueable{}
	q.mtx.Unlock()
	return nil
}

func (q *queue) Dequeue(num int) ([]Queueable, error) {
	if num == 0 {
		return nil, nil
	}

	q.mtx.Lock()
	defer q.mtx.Unlock()

	actual_num := min(len(q.queueables), num)
	dequeued := q.queueables[:actual_num]
	q.queueables = q.queueables[actual_num:]

	return dequeued, nil
}

func (q *queue) Enqueue(queueable Queueable) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if q.contains(queueable.Key()) {
		return AlreadyQueuedError.New("")
	}
	q.queueables = append(q.queueables, queueable)

	return nil
}

func (q *queue) List() ([]Queueable, error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.queueables, nil
}

func (q *queue) Position(key string) int {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	for idx, candidate := range q.queueables {
		if candidate.Key() == key {
			return idx + 1
		}
	}

	return -1
}

func (q *queue) Remove(key string) (queueable Queueable, err error) {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	if !q.contains(key) {
		return nil, NotFoundError.New("")
	}

	var removed Queueable
	without := []Queueable{}
	for _, candidate := range q.queueables {
		if key == candidate.Key() {
			removed = candidate
		} else {
			without = append(without, candidate)
		}
	}
	q.queueables = without

	return removed, nil
}

func (q *queue) Size() int {
	q.mtx.Lock()
	defer q.mtx.Unlock()

	return len(q.queueables)
}

// The caller is responsible for obtaining q.mtx if desired before calling
func (q *queue) contains(key string) bool {
	for _, candidate := range q.queueables {
		if key == candidate.Key() {
			return true
		}
	}

	return false
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
