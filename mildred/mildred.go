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

package mildred

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/fhs/gompd/mpd"
	"github.com/spacemonkeygo/spacelog"
)

var (
	mpdHost = flag.String("mildred.mpd_host",
		parseMpdHost(os.Getenv("MPD_HOST")),
		"MPD host for mildred")
	mpdPort = flag.String("mildred.mpd_port",
		parseMpdPort(os.Getenv("MPD_PORT")),
		"MPD port for mildred")
	mpdPassword = flag.String("mildred.mpd_password",
		parseMpdPassword(os.Getenv("MPD_HOST")),
		"MPD password for mildred (optional)")

	logger = spacelog.GetLogger()
)

type mpdConn struct {
	host     string
	port     string
	password string
}

type song struct {
	title        string
	artist       string
	album        string
	album_artist string
	time         int
}

func (s *song) Title() string {
	return s.title
}

func (s *song) Artist() string {
	return s.artist
}

func (s *song) Album() string {
	return s.album
}

func (s *song) AlbumArtist() string {
	return s.album_artist
}

func (s *song) Time() int {
	return s.time
}

func (s *song) String() string {
	return fmt.Sprintf("%q by %s from _%s_", s.title, s.artist, s.album)
}

func New() *mpdConn {
	logger.Debugf("mpd host: %q port: %q password: %t",
		*mpdHost, *mpdPort, *mpdPassword != "")
	return &mpdConn{
		host:     *mpdHost,
		port:     *mpdPort,
		password: *mpdPassword,
	}
}

func parseMpdHost(host string) string {
	pieces := strings.SplitN(host, "@", 2)
	if len(pieces) > 1 {
		return pieces[1]
	}

	if host == "" {
		return "localhost"
	}

	return host
}

func parseMpdPort(port string) string {
	if port == "" {
		return "6600"
	}

	return port
}

func parseMpdPassword(host string) string {
	pieces := strings.SplitN(host, "@", 2)
	if len(pieces) > 1 {
		return pieces[0]
	}

	return ""
}

func (m *mpdConn) CurrentSong() Song {
	conn, err := mpd.DialAuthenticated("tcp",
		net.JoinHostPort(m.host, m.port), m.password)
	if err != nil {
		logger.Errore(err)
		return nil
	}
	defer conn.Close()

	attrs, err := conn.CurrentSong()
	if err != nil {
		logger.Errore(err)
		return nil
	}
	logger.Debugf("attrs: %+v", attrs)

	return parseSong(attrs)
}

func parseSong(attrs mpd.Attrs) Song {
	time, err := strconv.ParseInt(attrs["Time"], 10, 32)
	if err != nil {
		logger.Warne(err)
		time = 0
	}

	return &song{
		title:        attrs["Title"],
		artist:       attrs["Artist"],
		album:        attrs["Album"],
		album_artist: attrs["AlbumArtist"],
		time:         int(time),
	}
}
