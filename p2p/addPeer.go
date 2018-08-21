package p2p

import (
	"context"
	"fmt"
	"log"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// pattern: /protocol-name/request-or-response-message/version
const addPeerRequest = "/addPeer/request/0.0.1"
const addPeerResponse = "/addPeer/response/0.0.1"

// AddPeerProtocol type
type AddPeerProtocol struct {
	node *Node     // local host
	done chan bool // only for demo purposes to stop main from terminating
}

func parseAddr(addrString string) (peerID peer.ID, protocolAddr ma.Multiaddr) {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(addrString) // ipfsaddr=/ip4/127.0.0.1/tcp/10000/ipfs/QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS) // pid=QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid) // peerid=<peer.ID VmDaab>
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)),
	)
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	return peerid, targetAddr
}

func NewAddPeerProtocol(node *Node) *AddPeerProtocol {
	p := &AddPeerProtocol{
		node: node,
		done: make(chan bool),
	}
	node.SetStreamHandler(addPeerRequest, p.onRequest)
	node.SetStreamHandler(addPeerResponse, p.onResponse)
	return p
}

// remote peer requests handler
func (p *AddPeerProtocol) onRequest(s inet.Stream) {
	// get request data
	data := makedataRequest()
	if !readBytes(data, s) {
		s.Close()
		return
	}

	log.Printf(
		"%s: Received addPeer request from %s. Message: %s",
		p.node.Name(),
		s.Conn().RemotePeer(),

	)

	// generate response message
	log.Printf(
		"%s: Sending addPeer response to %s",
		p.node.Name(),
		s.Conn().RemotePeer(),
	)

	resp := makeDataResponse()

	p.node.Peerstore().AddAddr(
		s.Conn().RemotePeer(),
		s.Conn().RemoteMultiaddr(),
		pstore.PermanentAddrTTL,
	)

	sResponse, err := p.node.NewStream(context.Background(), s.Conn().RemotePeer(), addPeerResponse)
	if err != nil {
		log.Println(err)
		return
	}
	if !sendBytes(resp, sResponse) {
		sResponse.Close()
	}
}

// remote addPeer response handler
func (p *AddPeerProtocol) onResponse(s inet.Stream) {
	data := makeDataResponse()
	if !readBytes(data, s) {
		s.Close()
		return
	}
	log.Printf(
		"%s: Received addPeer response from %s, result=%v",
		p.node.Name(),
		s.Conn().RemotePeer(),
	)
	p.done <- true
}

func (p *AddPeerProtocol) AddPeer(peerAddr string) bool {
	peerid, targetAddr := parseAddr(peerAddr)
	log.Printf("%s: Sending addPeer to: %s....", p.node.Name(), peerid)
	p.node.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	// create message data
	req := makedataRequest()

	s, err := p.node.NewStream(context.Background(), peerid, addPeerRequest)
	if err != nil {
		log.Println(err)
		return false
	}

	return sendBytes(req, s)
}
