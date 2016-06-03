// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

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
