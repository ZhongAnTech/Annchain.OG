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
package rpc

import (
	"context"
	"fmt"
	"github.com/annchain/OG/common/goroutine"
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
	C      *RpcController
}

func NewRpcServer(port string) *RpcServer {
	c := RpcController{}
	router := c.NewRouter()
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

func (srv *RpcServer) loggerInit() {
	//gin.DisableConsoleColor()
	// Logging to a file.
	//gin.DefaultWriter = io.MultiWriter(logrus.StandardLogger().Out)
	//logger := gin.LoggerWithWriter(logrus.StandardLogger().Out, "/")
	//srv.router.Use(gin.Logger())
}

func (srv *RpcServer) Start() {
	srv.loggerInit()
	logrus.Infof("listening Http on %s", srv.port)
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
	return fmt.Sprintf("rpcServer at port %s", srv.port)
}
