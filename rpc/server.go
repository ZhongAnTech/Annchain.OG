package rpc

import (
	"net/http"
	"github.com/sirupsen/logrus"
)

type RpcServer struct {
	rpcport string
}

func NewRpcServer(rpcport string) *RpcServer {
	srv := &RpcServer{
		rpcport: rpcport,
	}
	return srv
}

func (srv *RpcServer) Start() {
	// http.HandleFunc("/blocks", handleBlocks)
	// http.HandleFunc("/add_peer", handleAddPeer)
	go func() {
		logrus.Info("Listening HTTP on ", srv.rpcport)
		http.ListenAndServe(srv.rpcport, nil)
	}()
}
func (server *RpcServer) Stop() {
	logrus.Info("RPC Stopped")
}
