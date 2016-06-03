// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package json

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"xmtp.net/xmtpbot/seen"
	"xmtp.net/xmtpbot/seen/memory"

	"github.com/spacemonkeygo/spacelog"
)

const (
	dirPerms  = 0700
	filePerms = 0600
)

var (
	logger = spacelog.GetLogger()
)

type entry struct {
	Name string     `json:"name"`
	At   *time.Time `json:"when"`
}

type store struct {
	filename string
	mem      seen.Store
	mtx      sync.Mutex
}

func New(filename string) seen.Store {
	s := store{
		filename: filename,
		mem:      memory.New(),
	}
	if err := s.load(); err != nil {
		logger.Warnf("falling back to memory store: %v", err)
		return memory.New()
	}
	logger.Infof("seen store initialized at: %q", filename)

	return &s
}

func (s *store) MarkSeen(name string, at *time.Time) (err error) {
	s.mtx.Lock()
	err = s.mem.MarkSeen(name, at)
	s.mtx.Unlock()
	if err != nil {
		return err
	}

	return s.flush()
}

func (s *store) LastSeen(name string) (at *time.Time, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.mem.LastSeen(name)
}

func (s *store) Iterate(f func(name string, at *time.Time)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.mem.Iterate(f)
}

func (s *store) Length() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.mem.Length()
}

func (s *store) Idle(name string) (since *time.Duration, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return s.mem.Idle(name)
}

func (s *store) flush() (err error) {
	var links []entry
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.mem.Iterate(func(name string, at *time.Time) {
		links = append(links, entry{
			Name: name,
			At:   at,
		})
	})

	bytes, err := json.Marshal(links)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.filename, bytes, filePerms)
}

func (s *store) load() (err error) {
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

	var entries []entry
	err = json.Unmarshal(bytes, &entries)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		err = s.mem.MarkSeen(entry.Name, entry.At)
		if err != nil {
			logger.Errore(err)
			continue
		}
	}
	logger.Debugf("loaded %d seen records from JSON store", s.mem.Length())

	return nil
}

func (s *store) init() (err error) {
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
