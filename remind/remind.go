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

package remind

import (
	"encoding/hex"
	"math/rand"
	"sync"
	"time"
)

const (
	keyLength    = 64
	tickDuration = 5 * time.Second
)

type Remind interface {
	Set(at time.Time, fn func()) error
}

type remind struct {
	reminders map[string]reminder
	mtx       sync.Mutex
}

type reminder struct {
	at time.Time
	fn func()
}

func New() *remind {
	r := remind{
		reminders: make(map[string]reminder),
	}

	// Might be better to let the caller do this?
	go r.run()

	return &r
}

func (r *remind) run() {
	for {
		r.mtx.Lock()
		reminders := r.reminders
		r.mtx.Unlock()
		now := time.Now()

		for key, rem := range reminders {
			if rem.at.Before(now) {
				r.mtx.Lock()
				delete(r.reminders, key)
				r.mtx.Unlock()
				rem.fn()
			}
		}

		time.Sleep(tickDuration)
	}
}

func (r *remind) Set(at time.Time, fn func()) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.reminders[r.newKey()] = reminder{
		at: at,
		fn: fn,
	}

	return nil
}

func (r *remind) newKey() string {
	key := make([]byte, keyLength)
	rand.Read(key)

	return hex.EncodeToString(key)
}
