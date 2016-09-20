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

import "github.com/spacemonkeygo/errors"

var (
	Error              = errors.NewClass("queue")
	AlreadyQueuedError = Error.NewClass("user is already queued",
		errors.NoCaptureStack())
	NotFoundError = Error.NewClass("user is not queued",
		errors.NoCaptureStack())
)

type Queue interface {
	// Empty the Queue
	Clear() error

	// Return (at most) the first +n+ Queueables from the Queue
	Dequeue(n int) ([]Queueable, error)

	// Enqueue the given Queueable
	Enqueue(queueable Queueable) error

	// Return all Queueables in the Queue
	List() ([]Queueable, error)

	// Return the (1-indexed) position of the Queueable in the Queue
	//
	// Returns -1 if the key isn't found in the Queue.
	Position(key string) int

	// Remove the first Queueable found with a matching key from the Queue
	Remove(key string) error

	// Return the size of the Queue
	Size() int
}

type Queueable interface {
	// A key that's unique to the Queueables in a given Queue
	Key() string
}

type Manager interface {
	Lookup(channel_key string) Queue
}
