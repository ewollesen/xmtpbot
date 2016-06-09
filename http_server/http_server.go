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

package http_server

import (
	"flag"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"
)

var (
	address = flag.String("http.address", ":8080",
		"address to listen for HTTP api")

	logger = spacelog.GetLogger()

	Error = errors.NewClass("http_server")
)

type Server interface {
	GiveRouter(prefix string, fn func(*mux.Router) error) error
	Serve()
}

type server struct {
	mux *mux.Router
}

func New() Server {
	s := server{mux: mux.NewRouter()}

	s.mux.StrictSlash(true)
	s.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		logger.Debugf("%+v", req)
		w.Write([]byte("xMTP bot HTTP interface\n"))
	})

	return &s
}

func (s *server) Serve() {
	logger.Infof("listening on %s", *address)
	http.ListenAndServe(*address, s.mux)
}

func (s *server) GiveRouter(prefix string, fn func(*mux.Router) error) error {
	if len(prefix) == 0 || prefix == "/" {
		return Error.New("prefix must have length > 0")
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	return fn(s.mux.PathPrefix(prefix).Subrouter())
}
