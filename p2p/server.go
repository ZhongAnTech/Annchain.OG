package p2p

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"fmt"
)

const (
	queryBlock     = iota
	queryAll
	respQueryBlock
	respQueryChain
)

type P2PServer struct {
	port  string
	peers []*Peer
}

func NewP2PServer(port string) *P2PServer {
	srv := &P2PServer{
		port:  port,
		peers: []*Peer{},
	}
	return srv
}

func (srv *P2PServer) Start() {
	for _, peer := range srv.peers {
		go srv.readloop(peer.conn)
	}

	http.Handle("/", websocket.Handler(srv.readloop))
	log.Info("Listening p2p on ", srv.port)
	go http.ListenAndServe("127.0.0.1:"+srv.port, nil)
}

func (srv *P2PServer) Name() string {
	return fmt.Sprintf("P2PServer at port %s", srv.port)
}

func (srv *P2PServer) Close() {
	for _, peer := range srv.peers {
		peer.Close()
	}
}

func (srv *P2PServer) AddPeers(peerAddrs []string) {
	for _, peerAddr := range peerAddrs {
		if peerAddr == "" {
			continue
		}
		peer, err := NewPeer(peerAddr)
		if err != nil {
			continue
		}
		srv.peers = append(srv.peers, peer)
		go srv.readloop(peer.conn)
	}
}

func (srv *P2PServer) RemovePeer(peerAddr string) {
	for i, p := range srv.peers {
		if p.PeerAddr == peerAddr {
			p.Close()
			srv.peers = append(srv.peers[0:i], srv.peers[i+1:]...)
			return
		}
	}
}

type ResponseBlockchain struct {
	Type int    `json:"type"`
	Data []byte `json:"data"`
}

type QueryBlockMsg struct {
	Index int `json:"index"`
}

func (srv *P2PServer) readloop(ws *websocket.Conn) {
	var (
		v    = &ResponseBlockchain{}
		peer = ws.LocalAddr().String()
	)

	for {
		var msg []byte
		err := websocket.Message.Receive(ws, &msg)
		if err == io.EOF {
			log.Infof("p2p Peer[%s] shutdown, remove it form peers pool.\n", peer)
			break
		}
		if err != nil {
			log.WithError(err).Infof("Can't receive p2p msg from %s", peer)
			break
		}

		log.Infof("Received[from %s]: %s.\n", peer, msg)
		err = json.Unmarshal(msg, v)
		errFatal("invalid p2p msg", err)

		switch v.Type {
		// case queryBlock:
		// 	var queryData QueryBlockMsg
		// 	err = json.Unmarshal([]byte(v.Data), &queryData)
		// 	if err != nil {
		// 		log.Println("Can't unmarshal QueryBlockMsg from ", peer, err.Error())
		// 		continue
		// 	}
		// 	block := srv.blockchain.GetBlock(queryData.Index)
		// 	if block == nil {
		// 		log.Println("GetBlock with wrong index")
		// 		continue
		// 	}
		// 	v.Type = respQueryBlock
		// 	v.Data = block.Bytes()
		// 	bs, _ := json.Marshal(v)
		// 	log.Printf("responseLatestMsg: %s\n", bs)
		// 	ws.Write(bs)

		// case queryAll:
		// 	// TODO query all
		// 	v.Type = respQueryChain
		// 	v.Data = srv.blockchain.Bytes()
		// 	bs, _ := json.Marshal(v)
		// 	log.Printf("responseChainMsg: %s\n", bs)
		// 	ws.Write(bs)

		// case respQueryBlock:
		// 	// handleBlockchainResponse([]byte(v.Data))
		// 	var block types.Block
		// 	err = json.Unmarshal(v.Data, &block)
		// 	if err != nil {
		// 		log.Println("Can't unmarshal msg.Data to Block")
		// 		continue
		// 	}
		// 	srv.blockchain.TryAppendBlock(&block)

		// case respQueryChain:
		// 	// TODO
		}

	}
}
func (srv *P2PServer) Stop() {
	log.Info("P2P Stopped")
}

// ----------

func errFatal(msg string, err error) {
	if err != nil {
		log.WithError(err).Fatal(msg)
	}
}
