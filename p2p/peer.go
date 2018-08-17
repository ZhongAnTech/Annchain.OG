package p2p

import (
	log "github.com/sirupsen/logrus"

	"golang.org/x/net/websocket"
)

type Peer struct {
	PeerAddr string
	conn     *websocket.Conn
}

func NewPeer(peerAddr string) (*Peer, error) {
	ws, err := websocket.Dial(peerAddr, "", peerAddr)
	if err != nil {
		log.Println("dial to peer", peerAddr)
		return nil, err
	}
	p := &Peer{
		PeerAddr: peerAddr,
		conn:     ws,
	}
	return p, nil
}

func (p *Peer) Close() error {
	return p.conn.Close()
}
