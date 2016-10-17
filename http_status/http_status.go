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

package http_status

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/spacelog"

	"xmtp.net/xmtpbot/http_server"
)

var (
	Error  = errors.NewClass("http_status")
	logger = spacelog.GetLogger()
)

type StatusHandlerFunc func() map[string]string

type Status interface {
	Register(name string, handler StatusHandlerFunc)
}

type http_status struct {
	http_server http_server.Server
	handlers    map[string]StatusHandlerFunc
	up_since    time.Time
}

func New(http_server http_server.Server) *http_status {
	return &http_status{
		http_server: http_server,
		handlers:    make(map[string]StatusHandlerFunc),
		up_since:    time.Now(),
	}
}

func (s *http_status) Register(name string, handler StatusHandlerFunc) {
	s.handlers[name] = handler
	logger.Infof("added status handler: %q", name)
}

func (s *http_status) Run(shutdown chan bool, wg *sync.WaitGroup) (err error) {
	logger.Info("online")

	err = s.http_server.GiveRouter("status", s.ReceiveRouter)
	if err != nil {
		logger.Errore(err)
	}

	return nil
}

func (s *http_status) ReceiveRouter(router *mux.Router) (err error) {
	router.HandleFunc("/", s.handleHTTP)
	return nil
}

func (s *http_status) handleHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<h1>Status</h1>\n"))
	w.Write([]byte(fmt.Sprintf("<div><pre>%-20s %s\n</pre></div>\n",
		"uptime:", time.Since(s.up_since))))

	for name, handler := range s.handlers {
		w.Write([]byte(fmt.Sprintf("<h2>%s</h2>\n", name)))
		w.Write([]byte("<div><pre>\n"))
		for k, v := range handler() {
			w.Write([]byte(fmt.Sprintf("%-20s %s\n", k+":", v)))
		}
		w.Write([]byte("</pre></div>\n"))
	}
}
