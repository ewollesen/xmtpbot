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

package store

import "sync"

type memStore struct {
	mem map[string]string
	mtx sync.Mutex
}

func NewMemory() Simple {
	return &memStore{
		mem: make(map[string]string),
	}
}

func (mem *memStore) Get(key string) (value string, err error) {
	mem.mtx.Lock()
	defer mem.mtx.Unlock()

	return mem.mem[key], nil
}

func (mem *memStore) Set(key, value string) (err error) {
	mem.mtx.Lock()
	defer mem.mtx.Unlock()

	mem.mem[key] = value

	return nil
}

func (mem *memStore) Del(key string) (err error) {
	mem.mtx.Lock()
	defer mem.mtx.Unlock()

	delete(mem.mem, key)

	return nil
}

func (mem *memStore) Iterate(fn func(key, value string)) {
	mem.mtx.Lock()
	copy := mem.mem
	defer mem.mtx.Unlock()

	for key, value := range copy {
		fn(key, value)
	}
}
