// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package discv5

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p/nat"
	"github.com/annchain/OG/p2p/netutil"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

const Version = 4

// Errors
var (
	errPacketTooSmall = errors.New("too small")
	errBadPrefix      = errors.New("bad prefix")
	errTimeout        = errors.New("RPC timeout")
)

// Timeouts
const (
	respTimeout = 500 * time.Millisecond
	expiration  = 20 * time.Second

	driftThreshold = 10 * time.Second // Allowed clock drift before warning user
)

// RPC request structures

type CommonHash [32]byte

type (
	Ping struct {
		Version    uint
		From, To   RpcEndpoint
		Expiration uint64

		// v5
		//Topics []Topic
		Topics []string
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	// pong is the reply to ping.
	Pong struct {
		// This field should mirror the UDP envelope address
		// of the ping packet, which provides a way to discover the
		// the external address (after NAT).
		To RpcEndpoint

		ReplyTok   []byte // This contains the hash of the ping packet.
		Expiration uint64 // Absolute timestamp at which the packet becomes invalid.

		// v5
		TopicHash    CommonHash
		TicketSerial uint32
		WaitPeriods  []uint32

		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	// findnode is a query for nodes close to the given target.
	Findnode struct {
		//Target     NodeID // doesn't need to be an actual public key
		Target     [64]byte
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	// findnode is a query for nodes close to the given target.
	FindnodeHash struct {
		//Target     types.Hash
		Target     CommonHash
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	// reply to findnode
	Neighbors struct {
		Nodes      []RpcNode
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	TopicRegister struct {
		//Topics []Topic
		Topics []string
		Idx    uint
		Pong   []byte
	}

	TopicQuery struct {
		//Topic      Topic
		Topic      string
		Expiration uint64
	}

	// reply to topicQuery
	TopicNodes struct {
		//Echo  types.Hash
		Echo  CommonHash
		Nodes []RpcNode
	}

	RpcNode struct {
		//IP  net.IP // len 4 for IPv4 or 16 for IPv6
		IP  []byte
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
		//ID  NodeID
		ID [64]byte
	}

	RpcEndpoint struct {
		//IP  net.IP // len 4 for IPv4 or 16 for IPv6
		IP  []byte
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
	}
)

var (
	versionPrefix     = []byte("temporary discovery v5")
	versionPrefixSize = len(versionPrefix)
	sigSize           = 520 / 8
	headSize          = versionPrefixSize + sigSize // space of packet frame data
)

// Neighbors replies are sent across multiple packets to
// stay below the 1280 byte limit. We compute the maximum number
// of entries by stuffing a packet until it grows too large.
var maxNeighbors = func() int {
	p := Neighbors{Expiration: ^uint64(0)}
	maxSizeNode := RpcNode{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, maxSizeNode)
		data, err := p.MarshalMsg(nil)
		//size, _, err := rlp.EncodeToReader(p)
		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("cannot encode: " + err.Error())
		}
		size := len(data)
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()

var maxTopicNodes = func() int {
	p := TopicNodes{}
	maxSizeNode := RpcNode{IP: make(net.IP, 16), UDP: ^uint16(0), TCP: ^uint16(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, maxSizeNode)
		//size, _, err := rlp.EncodeToReader(p)
		_, err := p.MarshalMsg(nil)

		if err != nil {
			// If this ever happens, it will be caught by the unit tests.
			panic("cannot encode: " + err.Error())
		}
		size := p.Msgsize()
		if headSize+size+1 >= 1280 {
			return n
		}
	}
}()

func makeEndpoint(addr *net.UDPAddr, tcpPort uint16) RpcEndpoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndpoint{IP: ip, UDP: uint16(addr.Port), TCP: tcpPort}
}

func (e1 RpcEndpoint) equal(e2 RpcEndpoint) bool {
	e1Ip := net.IP(e1.IP)
	return e1.UDP == e2.UDP && e1.TCP == e2.TCP && e1Ip.Equal(e2.IP)
}

func nodeFromRPC(sender *net.UDPAddr, rn RpcNode) (*Node, error) {
	if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
		return nil, err
	}
	n := NewNode(rn.ID, rn.IP, rn.UDP, rn.TCP)
	err := n.validateComplete()
	return n, err
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{ID: n.ID, IP: n.IP, UDP: n.UDP, TCP: n.TCP}
}

type ingressPacket struct {
	remoteID   NodeID
	remoteAddr *net.UDPAddr
	ev         nodeEvent
	hash       []byte
	data       interface{} // one of the RPC structs
	rawData    []byte
}

type conn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// udp implements the RPC protocol.
type udp struct {
	conn        conn
	priv        *ecdsa.PrivateKey
	ourEndpoint RpcEndpoint
	nat         nat.Interface
	net         *Network
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(priv *ecdsa.PrivateKey, conn conn, realaddr *net.UDPAddr, nodeDBPath string, netrestrict *netutil.Netlist) (*Network, error) {
	transport, err := listenUDP(priv, conn, realaddr)
	if err != nil {
		return nil, err
	}
	net, err := newNetwork(transport, priv.PublicKey, nodeDBPath, netrestrict)
	if err != nil {
		return nil, err
	}
	log.WithField("net", net.tab.self).Info("UDP listener up")
	transport.net = net
	go transport.readLoop()
	return net, nil
}

func listenUDP(priv *ecdsa.PrivateKey, conn conn, realaddr *net.UDPAddr) (*udp, error) {
	return &udp{conn: conn, priv: priv, ourEndpoint: makeEndpoint(realaddr, uint16(realaddr.Port))}, nil
}

func (t *udp) localAddr() *net.UDPAddr {
	return t.conn.LocalAddr().(*net.UDPAddr)
}

func (t *udp) Close() {
	t.conn.Close()
}

func (t *udp) send(remote *Node, ptype nodeEvent, data []byte) (hash []byte) {
	hash, _ = t.sendPacket(remote.ID, remote.addr(), byte(ptype), data)
	return hash
}

func (t *udp) sendPing(remote *Node, toaddr *net.UDPAddr, topics []Topic) (hash []byte) {
	ping := Ping{
		Version:    Version,
		From:       t.ourEndpoint,
		To:         makeEndpoint(toaddr, uint16(toaddr.Port)), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
		Topics:     topicsToStrings(topics),
	}
	data, _ := ping.MarshalMsg(nil)
	hash, _ = t.sendPacket(remote.ID, toaddr, byte(pingPacket), data)
	return hash
}

func (t *udp) sendFindnode(remote *Node, target NodeID) {
	f := Findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	data, _ := f.MarshalMsg(nil)
	t.sendPacket(remote.ID, remote.addr(), byte(findnodePacket), data)
}

func (t *udp) sendNeighbours(remote *Node, results []*Node) {
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	p := Neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	for i, result := range results {
		p.Nodes = append(p.Nodes, nodeToRPC(result))
		if len(p.Nodes) == maxNeighbors || i == len(results)-1 {
			data, _ := p.MarshalMsg(nil)
			t.sendPacket(remote.ID, remote.addr(), byte(neighborsPacket), data)
			p.Nodes = p.Nodes[:0]
		}
	}
}

func (t *udp) sendFindnodeHash(remote *Node, target types.Hash) {
	f := FindnodeHash{
		Target:     target.Bytes,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	data, _ := f.MarshalMsg(nil)
	t.sendPacket(remote.ID, remote.addr(), byte(findnodeHashPacket), data)
}

func (t *udp) sendTopicRegister(remote *Node, topics []Topic, idx int, pong []byte) {
	tp := TopicRegister{
		Topics: topicsToStrings(topics),
		Idx:    uint(idx),
		Pong:   pong,
	}
	data, _ := tp.MarshalMsg(nil)
	t.sendPacket(remote.ID, remote.addr(), byte(topicRegisterPacket), data)
}

func (t *udp) sendTopicNodes(remote *Node, queryHash types.Hash, nodes []*Node) {
	p := TopicNodes{Echo: queryHash.Bytes}
	var sent bool
	for _, result := range nodes {
		IP := net.IP(result.IP)
		if IP.Equal(t.net.tab.self.IP) || netutil.CheckRelayIP(remote.IP, result.IP) == nil {
			p.Nodes = append(p.Nodes, nodeToRPC(result))
		}
		if len(p.Nodes) == maxTopicNodes {
			data, _ := p.MarshalMsg(nil)
			t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), data)
			p.Nodes = p.Nodes[:0]
			sent = true
		}
	}
	if !sent || len(p.Nodes) > 0 {
		data, _ := p.MarshalMsg(nil)
		t.sendPacket(remote.ID, remote.addr(), byte(topicNodesPacket), data)
	}
}

func (t *udp) sendPacket(toid NodeID, toaddr *net.UDPAddr, ptype byte, data []byte) (hash []byte, err error) {
	//fmt.Println("sendPacket", nodeEvent(ptype), toaddr.String(), toid.String())
	packet, hash, err := encodePacket(t.priv, ptype, data)
	if err != nil {
		//fmt.Println(err)
		return hash, err
	}
	log.Debugf(">>> %v to %x@%v", nodeEvent(ptype), toid[:8], toaddr)
	if _, err := t.conn.WriteToUDP(packet, toaddr); err != nil {
		log.WithError(err).Debug("UDP send failed")
	} else {
		//egressTrafficMeter.Mark(int64(nbytes))
	}
	//fmt.Println(err)
	return hash, err
}

// zeroed padding space for encodePacket.
var headSpace = make([]byte, headSize)

func encodePacket(priv *ecdsa.PrivateKey, ptype byte, data []byte) (p, hash []byte, err error) {
	b := new(bytes.Buffer)
	b.Write(headSpace)
	b.WriteByte(ptype)
	b.Write(data)
	packet := b.Bytes()
	sig, err := crypto.Sign(crypto.Keccak256(packet[headSize:]), priv)
	if err != nil {
		log.WithError(err).Error("could not sign packet")
		return nil, nil, err
	}
	copy(packet, versionPrefix)
	copy(packet[versionPrefixSize:], sig)
	hash = crypto.Keccak256(packet[versionPrefixSize:])
	return packet, hash, nil
}

// readLoop runs in its own goroutine. it injects ingress UDP packets
// into the network loop.
func (t *udp) readLoop() {
	defer t.conn.Close()
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't match.
	buf := make([]byte, 1280)
	for {
		nbytes, from, err := t.conn.ReadFromUDP(buf)
		//ingressTrafficMeter.Mark(int64(nbytes))
		if netutil.IsTemporaryError(err) {
			// Ignore temporary read errors.
			log.WithError(err).Debug("Temporary read error")
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			log.WithError(err).Debug("Read error")
			return
		}
		t.handlePacket(from, buf[:nbytes])
	}
}

func (t *udp) handlePacket(from *net.UDPAddr, buf []byte) error {
	pkt := ingressPacket{remoteAddr: from}
	if err := decodePacket(buf, &pkt); err != nil {
		log.WithError(err).WithField("from", from).Debug("Bad packet")
		//fmt.Println("bad packet", err)
		return err
	}
	t.net.reqReadPacket(pkt)
	return nil
}

func decodePacket(buffer []byte, pkt *ingressPacket) error {
	if len(buffer) < headSize+1 {
		return errPacketTooSmall
	}
	buf := make([]byte, len(buffer))
	copy(buf, buffer)
	prefix, sig, sigdata := buf[:versionPrefixSize], buf[versionPrefixSize:headSize], buf[headSize:]
	if !bytes.Equal(prefix, versionPrefix) {
		return errBadPrefix
	}
	fromID, err := recoverNodeID(crypto.Keccak256(buf[headSize:]), sig)
	if err != nil {
		return err
	}
	pkt.rawData = buf
	pkt.hash = crypto.Keccak256(buf[versionPrefixSize:])
	pkt.remoteID = fromID
	data := sigdata[1:]
	switch pkt.ev = nodeEvent(sigdata[0]); pkt.ev {
	case pingPacket:
		req := new(Ping)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case pongPacket:
		req := new(Pong)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case findnodePacket:
		req := new(Findnode)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case neighborsPacket:
		req := new(Neighbors)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case findnodeHashPacket:
		req := new(FindnodeHash)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case topicRegisterPacket:
		req := new(TopicRegister)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case topicQueryPacket:
		req := new(TopicQuery)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	case topicNodesPacket:
		req := new(TopicNodes)
		_, err = req.UnmarshalMsg(data)
		pkt.data = req
	default:
		return fmt.Errorf("unknown packet type: %d", sigdata[0])
	}
	//s := rlp.NewStream(bytes.NewReader(sigdata[1:]), 0)
	//err = s.Decode(pkt.data)
	return err
}
