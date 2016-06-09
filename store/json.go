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

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

const (
	dirPerms  = 0700
	filePerms = 0600
)

type jsonStore struct {
	filename string
	mem      map[string]string
	mtx      sync.Mutex
}

func New(filename string) Simple {
	s := jsonStore{
		filename: filename,
		mem:      make(map[string]string),
	}
	if err := s.load(); err != nil {
		logger.Warnf("failed to load existing state: %v", err)
	}
	logger.Infof("json simple store initialized at: %q", filename)

	return &s
}

func (s *jsonStore) Set(key, value string) (err error) {
	s.mtx.Lock()
	s.mem[key] = value
	s.mtx.Unlock()

	return s.flush()
}

func (s *jsonStore) Get(key string) (value string, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.mem[key], nil
}

func (s *jsonStore) Iterate(fn func(key, value string)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for key, value := range s.mem {
		fn(key, value)
	}
}

func (s *jsonStore) Del(key string) (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	delete(s.mem, key)

	return nil
}

func (s *jsonStore) flush() (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	bytes, err := json.MarshalIndent(s.mem, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.filename, bytes, filePerms)
}

func (s *jsonStore) load() (err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	bytes, err := ioutil.ReadFile(s.filename)
	if err != nil {
		if err2, ok := err.(*os.PathError); ok && err2.Op == "open" {
			return s.init()
		} else {
			return err
		}
	}
	// TODO handle empty file

	err = json.Unmarshal(bytes, &s.mem)
	if err != nil {
		return err
	}

	logger.Debugf("loaded %d seen records from JSON store", len(s.mem))

	return nil
}

func (s *jsonStore) init() (err error) {
	err = os.MkdirAll(filepath.Dir(s.filename), dirPerms)
	if err != nil {
		return err
	}
	_, err = os.OpenFile(s.filename, os.O_CREATE|os.O_APPEND, filePerms)
	if err != nil {
		return err
	}

	return nil
}
