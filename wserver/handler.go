// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
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
package wserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// websocketHandler defines to handle websocket upgrade request.
type websocketHandler struct {
	// upgrader is used to upgrade request.
	upgrader *websocket.Upgrader

	event2Cons *event2Cons
	baseConns  []*Conn
}

// RegisterMessage defines message struct client send after connect
// to the server.
type RegisterMessage struct {
	//Token string
	Event string `json:"event"`
}

func (wh *websocketHandler) Handle(ctx *gin.Context) {
	wh.ServeHTTP(ctx.Writer, ctx.Request)
}

// First try to upgrade connection to websocket. If success, connection will
// be kept until client send close message or server drop them.
func (wh *websocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsConn, err := wh.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer wsConn.Close()

	// handle Websocket request
	conn := NewConn(wsConn)
	var eventType string
	conn.AfterReadFunc = func(messageType int, r io.Reader) {
		var rm RegisterMessage
		decoder := json.NewDecoder(r)
		if err := decoder.Decode(&rm); err != nil {
			logrus.WithError(err).Debug("Failed to serve request")
			return
		}

		wh.event2Cons.Add(rm.Event, conn)
		eventType = rm.Event
	}
	conn.BeforeCloseFunc = func() {
		wh.event2Cons.Remove(eventType, conn)
	}
	wh.event2Cons.Add(messageTypeBaseWs, conn)

	conn.Listen()
}

// ErrRequestIllegal describes error when data of the request is unaccepted.
var ErrRequestIllegal = errors.New("request data illegal")

// pushHandler defines to handle push message request.
type pushHandler struct {
	// authFunc defines to authorize request. The request will proceed only
	// when it returns true.
	authFunc func(r *http.Request) bool

	event2Cons *event2Cons
}

func (s *pushHandler) Handle(ctx *gin.Context) {
	s.ServeHTTP(ctx.Writer, ctx.Request)
}

// Authorize if needed. Then decode the request and push message to each
// realted websocket connection.
func (s *pushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// authorize
	if s.authFunc != nil {
		if ok := s.authFunc(r); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	// read request
	var pm PushMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&pm); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(ErrRequestIllegal.Error()))
		return
	}

	// validate the data
	if pm.Event == "" || pm.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(ErrRequestIllegal.Error()))
		return
	}

	cnt, err := s.push(pm.Event, pm.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	result := strings.NewReader(fmt.Sprintf("message sent to %d clients", cnt))
	io.Copy(w, result)
}

func (s *pushHandler) push(event, message string) (int, error) {
	if event == "" || message == "" {
		return 0, errors.New("parameters(userId, event, message) can't be empty")
	}

	conns, err := s.event2Cons.Get(event)
	if err != nil {
		return 0, fmt.Errorf("Get conns with eventType: %s failed!\n", event)
	}
	cnt := 0
	for i := range conns {
		_, err := conns[i].Write([]byte(message))
		if err != nil {
			s.event2Cons.Remove(event, conns[i])
			continue
		}
		cnt++
	}

	return cnt, nil
}

// PushMessage defines message struct send by client to push to each connected
// websocket client.
type PushMessage struct {
	//UserID  string `json:"userId"`
	Event   string `json:"event"`
	Message string `json:"message"`
}
