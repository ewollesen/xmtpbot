// Copyright (C) 2016 Eric Wollesen <ericw at xmtp dot net>

package mildred

type Song interface {
	Title() string
	Artist() string
	Album() string
	AlbumArtist() string
	Time() int
	String() string
}

type Conn interface {
	CurrentSong() Song
}
