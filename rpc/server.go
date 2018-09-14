package rpc

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

const ShutdownTimeoutSeconds = 5

type RpcServer struct {
	router *gin.Engine
	server *http.Server
	port   string
	C      *RpcControler
}

func NewRpcServer(port string) *RpcServer {
	c := RpcControler{}
	router := c.Newrouter()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	rpc := &RpcServer{
		port:   port,
		router: router,
		server: server,
		C:      &c,
	}
	return rpc
}

func (srv *RpcServer) Start() {
	logrus.Infof("listening Http on %s", srv.port)
	go func() {
		// service connections
		if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatalf("error in Http server")
		}
	}()
}

func (srv *RpcServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeoutSeconds*time.Second)
	defer cancel()
	if err := srv.server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatalf("error while shutting down the Http server")
	}
	logrus.Infof("http server Stopped")
}

func (srv *RpcServer) Name() string {
	return fmt.Sprintf("rpcServer at port %s", srv.port)
}
