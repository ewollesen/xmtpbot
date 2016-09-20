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

import "sync"

type manager struct {
	queues map[string]Queue
	mtx    sync.Mutex
}

func NewManager() Manager {
	return &manager{
		queues: make(map[string]Queue),
	}
}

func (m *manager) Lookup(key string) Queue {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	q, ok := m.queues[key]
	if !ok {
		q = New()
		m.queues[key] = q
	}
	return q
}
