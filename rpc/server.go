package rpc

import (
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"time"
	"context"
	"fmt"
)

const ShutdownTimeoutSeconds = 5

type RpcServer struct {
	router *gin.Engine
	server *http.Server
	port   string
}

func NewRpcServer(port string) *RpcServer {

	router := gin.Default()

	// init paths here
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	rpc := &RpcServer{
		port:   port,
		router: router,
		server: server,
	}

	return rpc
}

func (srv *RpcServer) Start() {
	logrus.Infof("Listening Http on %s", srv.port)
	go func() {
		// service connections
		if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatalf("Error in Http server")
		}
	}()
}

func (srv *RpcServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeoutSeconds*time.Second)
	defer cancel()
	if err := srv.server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatalf("Error while shutting down the Http server")
	}
	logrus.Infof("Http server Stopped")
}

func (srv *RpcServer) Name() string {
	return fmt.Sprintf("RpcServer at port %s", srv.port)
}