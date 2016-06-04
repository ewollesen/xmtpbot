// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package store

type Simple interface {
	Get(key string) (value string, err error)
	Set(key, value string) (err error)
	Del(key string) (err error)
	Iterate(func(key, value string))
}
