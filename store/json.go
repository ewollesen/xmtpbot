// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

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
