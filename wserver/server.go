// Package wserver provides building simple websocket server with message push.
package wserver

import (
	"net/http"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
	"context"
	"github.com/annchain/OG/types"
)

const (
	serverDefaultWSPath   = "/ws"
	serverDefaultPushPath = "/push"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(*http.Request) bool {
		return true
	},
}

// Server defines parameters for running websocket server.
type Server struct {
	// Address for server to listen on
	Addr string

	// Path for websocket request, default "/ws".
	WSPath string

	// Path for push message, default "/push".
	PushPath string

	// Upgrader is for upgrade connection to websocket connection using
	// "github.com/gorilla/websocket".
	//
	// If Upgrader is nil, default upgrader will be used. Default upgrader is
	// set ReadBufferSize and WriteBufferSize to 1024, and CheckOrigin always
	// returns true.
	Upgrader *websocket.Upgrader

	// Authorize push request. Message will be sent if it returns true,
	// otherwise the request will be discarded. Default nil and push request
	// will always be accepted.
	PushAuth func(r *http.Request) bool

	// To receive new tx events
	NewTxReceivedChan chan types.Txi

	wh     *websocketHandler
	ph     *pushHandler
	engine *gin.Engine
	server *http.Server
	quit   chan bool
}

// ListenAndServe listens on the TCP network address and handle websocket
// request.
func (s *Server) Serve() {
	if err := s.server.ListenAndServe(); err != nil {
		// cannot panic, because this probably is an intentional close
		logrus.WithError(err).Info("websocket server")
	}
}

// Push filters connections by userID and event, then write message
func (s *Server) Push(event, message string) (int, error) {
	return s.ph.push(event, message)
}

// NewServer creates a new Server.
func NewServer(addr string) *Server {
	s := &Server{
		Addr:              addr,
		WSPath:            serverDefaultWSPath,
		PushPath:          serverDefaultPushPath,
		NewTxReceivedChan: make(chan types.Txi),
		quit:              make(chan bool),
	}

	e2c := NewEvent2Cons()

	// websocket request handler
	wh := websocketHandler{
		upgrader:   defaultUpgrader,
		event2Cons: e2c,
	}
	if s.Upgrader != nil {
		wh.upgrader = s.Upgrader
	}
	s.wh = &wh

	// push request handler
	ph := pushHandler{
		event2Cons: e2c,
	}
	if s.PushAuth != nil {
		ph.authFunc = s.PushAuth
	}
	s.ph = &ph

	engine := gin.Default()
	engine.GET(s.WSPath, wh.Handle)
	engine.GET(s.PushPath, ph.Handle)

	s.server = &http.Server{
		Addr:    s.Addr,
		Handler: engine,
	}
	return s
}

func (s *Server) Start() {
	go s.Serve()
	go s.WatchNewTxs()
}

func (s *Server) Stop() {
	s.quit <- true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Info("server Shutdown")
	}
	logrus.Info("server exiting")
}

func (s *Server) Name() string {
	return fmt.Sprintf("websocket Server at %s", s.Addr)
}
func (s *Server) WatchNewTxs() {
	for {
		select {
		case tx := <-s.NewTxReceivedChan:
			msg := tx2UIData(tx)
			logrus.WithField("msg", msg).Info("push to ws")
			s.Push("new_unit", msg)
		case <-s.quit:
			break
		}
	}

}
