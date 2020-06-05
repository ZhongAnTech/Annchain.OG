package rpc

import (
	"context"
	"fmt"
	"github.com/annchain/OG/common/goroutine"
	"github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"time"
)

const ShutdownTimeoutSeconds = 5

type RpcServer struct {
	Controller *RpcController
	Port       int
	server     *http.Server
}

func (srv *RpcServer) InitDefault() {
	router := srv.Controller.NewRouter()
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(srv.Port),
		Handler: router,
	}
	srv.server = server
}

func (srv *RpcServer) Start() {
	logrus.Infof("listening Http on %d", srv.Port)
	goroutine.New(func() {
		// service connections
		if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatalf("error in Http server")
		}
	})
}

func (srv *RpcServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeoutSeconds*time.Second)
	defer cancel()
	if err := srv.server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("error while shutting down the Http server")
	}
	logrus.Infof("http server Stopped")
}

func (srv *RpcServer) Name() string {
	return fmt.Sprintf("rpcServer at port %d", srv.Port)
}
