package rpc

import (
	log "github.com/sirupsen/logrus"
	"net/http"
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
		log.Println("Listen HTTP on", srv.rpcport)
		log.Fatalln("start api server", http.ListenAndServe(srv.rpcport, nil))
	}()
}
