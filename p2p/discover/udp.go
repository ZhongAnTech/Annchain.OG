// Copyright 2015 The go-ethereum Authors
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
//go:generate msgp
package discover

import (
	"bytes"
	"container/list"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/annchain/OG/p2p/enode"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p/netutil"
)

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errBadHash          = errors.New("bad hash")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNode      = errors.New("unknown node")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
)

// Timeouts
const (
	respTimeout    = 500 * time.Millisecond
	expiration     = 20 * time.Second
	bondExpiration = 24 * time.Hour

	ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold      = 10 * time.Second // Allowed clock drift before warning user
)

// RPC packet types
const (
	pingPacket = iota + 1 // zero is 'reserved'
	pongPacket
	findnodePacket
	neighborsPacket
)

// RPC request structures
type (
	Ping struct {
		Version    uint
		From, To   RpcEndpoint
		Expiration uint64
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
		// Ignore additional fields (for forward compatibility).
		Rest [][]byte `rlp:"tail"`
	}

	// findnode is a query for nodes close to the given target.
	Findnode struct {
		//Target     NodeID // doesn't need to be an actual public key
		Target     EncPubkey
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

	RpcNode struct {
		//IP  net.IP // len 4 for IPv4 or 16 for IPv6
		IP  []byte
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
		ID  EncPubkey
	}

	RpcEndpoint struct {
		//IP  net.IP // len 4 for IPv4 or 16 for IPv6
		IP  []byte
		UDP uint16 // for discovery protocol
		TCP uint16 // for RLPx protocol
	}
)

func (r RpcEndpoint) Equal(r1 RpcEndpoint) bool {
	if len(r.IP) != len(r1.IP) {
		return false
	}
	if !bytes.Equal(r.IP, r1.IP) {
		return false
	}
	if r.TCP != r1.TCP {
		return false
	}
	if r.UDP != r1.UDP {
		return false
	}
	return true
}

func makeEndpoint(addr *net.UDPAddr, tcpPort uint16) RpcEndpoint {
	ip := net.IP{}
	if ip4 := addr.IP.To4(); ip4 != nil {
		ip = ip4
	} else if ip6 := addr.IP.To16(); ip6 != nil {
		ip = ip6
	}
	return RpcEndpoint{IP: ip, UDP: uint16(addr.Port), TCP: tcpPort}
}

func (t *udp) nodeFromRPC(sender *net.UDPAddr, rn RpcNode) (*node, error) {
	if rn.UDP <= 1024 {
		return nil, errors.New("low port")
	}
	if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
		return nil, err
	}
	if t.netrestrict != nil && !t.netrestrict.Contains(rn.IP) {
		return nil, errors.New("not contained in netrestrict whitelist")
	}
	key, err := decodePubkey(rn.ID)
	if err != nil {
		return nil, err
	}
	n := wrapNode(enode.NewV4(key, rn.IP, int(rn.TCP), int(rn.UDP)))
	err = n.ValidateComplete()
	return n, err
}

func nodeToRPC(n *node) RpcNode {
	var key ecdsa.PublicKey
	var ekey EncPubkey
	if err := n.Load((*enode.Secp256k1)(&key)); err == nil {
		ekey = encodePubkey(&key)
	}
	return RpcNode{ID: ekey, IP: n.IP(), UDP: uint16(n.UDP()), TCP: uint16(n.TCP())}
}

type packet interface {
	handle(t *udp, from *net.UDPAddr, fromKey EncPubkey, mac []byte) error
	name() string
}

type conn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// udp implements the discovery v4 UDP wire protocol.
type udp struct {
	conn        conn
	netrestrict *netutil.Netlist
	priv        *ecdsa.PrivateKey
	localNode   *enode.LocalNode
	db          *enode.DB
	tab         *Table
	wg          sync.WaitGroup

	addpending chan *pending
	gotreply   chan reply
	closing    chan struct{}
}

// pending represents a pending reply.
//
// some implementations of the protocol wish to send more than one
// reply packet to findnode. in general, any neighbors packet cannot
// be matched up with a specific findnode packet.
//
// our implementation handles this by storing a callback function for
// each pending reply. incoming packets from a node are dispatched
// to all the callback functions for that node.
type pending struct {
	// these fields must match in the reply.
	from  enode.ID
	ptype byte

	// time when the request must complete
	deadline time.Time

	// callback is called when a matching reply arrives. if it returns
	// true, the callback is removed from the pending reply queue.
	// if it returns false, the reply is considered incomplete and
	// the callback will be invoked again for the next matching reply.
	callback func(resp interface{}) (done bool)

	// errc receives nil when the callback indicates completion or an
	// error if no further reply is received within the timeout.
	errc chan<- error
}

type reply struct {
	from  enode.ID
	ptype byte
	data  interface{}
	// loop indicates whether there was
	// a matching request by sending on this channel.
	matched chan<- bool
}

// ReadPacket is sent to the unhandled channel when it could not be processed
type ReadPacket struct {
	Data []byte       `msg:"-"`
	Addr *net.UDPAddr `msg:"-"`
}

// Config holds Table-related settings.
type Config struct {
	// These settings are required and configure the UDP listener:
	PrivateKey *ecdsa.PrivateKey `msg:"-"`

	// These settings are optional:
	NetRestrict *netutil.Netlist  `msg:"-"` // network whitelist
	Bootnodes   []*enode.Node     `msg:"-"` // list of bootstrap nodes
	Unhandled   chan<- ReadPacket `msg:"-"` // unhandled packets are sent on this channel
}

// ListenUDP returns a new table that listens for UDP packets on laddr.
func ListenUDP(c conn, ln *enode.LocalNode, cfg Config) (*Table, error) {
	tab, _, err := newUDP(c, ln, cfg)
	if err != nil {
		return nil, err
	}
	return tab, nil
}

func newUDP(c conn, ln *enode.LocalNode, cfg Config) (*Table, *udp, error) {
	udp := &udp{
		conn:        c,
		priv:        cfg.PrivateKey,
		netrestrict: cfg.NetRestrict,
		localNode:   ln,
		db:          ln.Database(),
		closing:     make(chan struct{}),
		gotreply:    make(chan reply),
		addpending:  make(chan *pending),
	}
	tab, err := newTable(udp, ln.Database(), cfg.Bootnodes)
	if err != nil {
		return nil, nil, err
	}
	udp.tab = tab

	udp.wg.Add(2)
	go udp.loop()
	go udp.readLoop(cfg.Unhandled)
	return udp.tab, udp, nil
}

func (t *udp) self() *enode.Node {
	return t.localNode.Node()
}

func (t *udp) close() {
	close(t.closing)
	t.conn.Close()
	t.wg.Wait()
}
func (t *udp) ourEndpoint() RpcEndpoint {
	n := t.self()
	a := &net.UDPAddr{IP: n.IP(), Port: n.UDP()}
	return makeEndpoint(a, uint16(n.TCP()))
}

// ping sends a ping message to the given node and waits for a reply.
func (t *udp) ping(toid enode.ID, toaddr *net.UDPAddr) error {
	return <-t.sendPing(toid, toaddr, nil)
}

// sendPing sends a ping message to the given node and invokes the callback
// when the reply arrives.
func (t *udp) sendPing(toid enode.ID, toaddr *net.UDPAddr, callback func()) <-chan error {
	req := &Ping{
		Version:    4,
		From:       t.ourEndpoint(),
		To:         makeEndpoint(toaddr, 0), // TODO: maybe use known TCP port from DB
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	data, _ := req.MarshalMsg(nil)
	packet, hash, err := encodePacket(t.priv, pingPacket, data)
	if err != nil {
		errc := make(chan error, 1)
		errc <- err
		return errc
	}
	errc := t.pending(toid, pongPacket, func(p interface{}) bool {
		ok := bytes.Equal(p.(*Pong).ReplyTok, hash)
		if ok && callback != nil {
			callback()
		}
		return ok
	})
	t.localNode.UDPContact(toaddr)
	t.write(toaddr, req.name(), packet)
	return errc
}

func (t *udp) waitping(from enode.ID) error {
	return <-t.pending(from, pingPacket, func(interface{}) bool { return true })
}

// findnode sends a findnode request to the given node and waits until
// the node has sent up to k neighbors.
func (t *udp) findnode(toid enode.ID, toaddr *net.UDPAddr, target EncPubkey) ([]*node, error) {
	// If we haven't seen a ping from the destination node for a while, it won't remember
	// our endpoint proof and reject findnode. Solicit a ping first.
	if time.Since(t.db.LastPingReceived(toid)) > bondExpiration {
		t.ping(toid, toaddr)
		t.waitping(toid)
	}

	nodes := make([]*node, 0, bucketSize)
	nreceived := 0
	errc := t.pending(toid, neighborsPacket, func(r interface{}) bool {
		reply := r.(*Neighbors)
		for _, rn := range reply.Nodes {
			nreceived++
			n, err := t.nodeFromRPC(toaddr, rn)
			if err != nil {
				log.WithFields(logrus.Fields{"ip": rn.IP, "addr": toaddr, "udp": rn.UDP}).
					WithError(err).
					Trace("Invalid neighbor node received")
				continue
			}
			nodes = append(nodes, n)
		}
		return nreceived >= bucketSize
	})
	findNode := &Findnode{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	data, _ := findNode.MarshalMsg(nil)
	t.send(toaddr, findnodePacket, data, findNode.name())
	return nodes, <-errc
}

// pending adds a reply callback to the pending reply queue.
// see the documentation of type pending for a detailed explanation.
func (t *udp) pending(id enode.ID, ptype byte, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case t.addpending <- p:
		// loop will handle it
	case <-t.closing:
		ch <- errClosed
	}
	return ch
}

func (t *udp) handleReply(from enode.ID, ptype byte, req packet) bool {
	matched := make(chan bool, 1)
	select {
	case t.gotreply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-t.closing:
		return false
	}
}

// loop runs in its own goroutine. it keeps track of
// the refresh timer and the pending reply queue.
func (t *udp) loop() {
	defer t.wg.Done()

	var (
		plist        = list.New()
		timeout      = time.NewTimer(0)
		nextTimeout  *pending // head of plist when timeout was last reset
		contTimeouts = 0      // number of continuous timeouts to do NTP checks
		ntpWarnTime  = time.Unix(0, 0)
	)
	<-timeout.C // ignore first timeout
	defer timeout.Stop()

	resetTimeout := func() {
		if plist.Front() == nil || nextTimeout == plist.Front().Value {
			return
		}
		// Start the timer so it fires when the next pending reply has expired.
		now := time.Now()
		for el := plist.Front(); el != nil; el = el.Next() {
			nextTimeout = el.Value.(*pending)
			if dist := nextTimeout.deadline.Sub(now); dist < 2*respTimeout {
				timeout.Reset(dist)
				return
			}
			// Remove pending replies whose deadline is too far in the
			// future. These can occur if the system clock jumped
			// backwards after the deadline was assigned.
			nextTimeout.errc <- errClockWarp
			plist.Remove(el)
		}
		nextTimeout = nil
		timeout.Stop()
	}

	for {
		resetTimeout()

		select {
		case <-t.closing:
			for el := plist.Front(); el != nil; el = el.Next() {
				el.Value.(*pending).errc <- errClosed
			}
			return

		case p := <-t.addpending:
			p.deadline = time.Now().Add(respTimeout)
			plist.PushBack(p)

		case r := <-t.gotreply:
			var matched bool
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.from == r.from && p.ptype == r.ptype {
					matched = true
					// Remove the matcher if its callback indicates
					// that all replies have been received. This is
					// required for packet types that expect multiple
					// reply packets.
					if p.callback(r.data) {
						p.errc <- nil
						plist.Remove(el)
					}
					// Reset the continuous timeout counter (time drift detection)
					contTimeouts = 0
				}
			}
			if matched == false {
				log.WithField("fromId", r.from.TerminalString()).WithField(
					//"rfrom",r.fromAddr.String()).WithField(
					"rtype", r.ptype).Debug("mismatch")
			}
			r.matched <- matched

		case now := <-timeout.C:
			nextTimeout = nil

			// Notify and remove callbacks whose deadline is in the past.
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if now.After(p.deadline) || now.Equal(p.deadline) {
					p.errc <- errTimeout
					plist.Remove(el)
					contTimeouts++
				}
			}
			// If we've accumulated too many timeouts, do an NTP time sync check
			if contTimeouts > ntpFailureThreshold {
				if time.Since(ntpWarnTime) >= ntpWarningCooldown {
					ntpWarnTime = time.Now()
					go checkClockDrift()
				}
				contTimeouts = 0
			}
		}

	}
}

const (
	macSize  = 256 / 8
	sigSize  = 520 / 8
	headSize = macSize + sigSize // space of packet frame data
)

var (
	headSpace = make([]byte, headSize)

	// Neighbors replies are sent across multiple packets to
	// stay below the 1280 byte limit. We compute the maximum number
	// of entries by stuffing a packet until it grows too large.
	maxNeighbors int
)

func init() {
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
			maxNeighbors = n
			break
		}
	}
}

func (t *udp) send(toaddr *net.UDPAddr, ptype byte, data []byte, name string) ([]byte, error) {
	packet, hash, err := encodePacket(t.priv, ptype, data)
	if err != nil {
		return hash, err
	}
	return hash, t.write(toaddr, name, packet)
}

func (t *udp) write(toaddr *net.UDPAddr, what string, packet []byte) error {
	_, err := t.conn.WriteToUDP(packet, toaddr)
	log.WithField("addr", toaddr).WithError(err).Trace(">>write  " + what)
	return err
}

func encodePacket(priv *ecdsa.PrivateKey, ptype byte, data []byte) (packet, hash []byte, err error) {
	b := new(bytes.Buffer)
	b.Write(headSpace)
	b.WriteByte(ptype)
	b.Write(data)
	packet = b.Bytes()
	sig, err := crypto.Sign(crypto.Keccak256(packet[headSize:]), priv)
	if err != nil {
		log.WithError(err).Error("Can't sign discv4 packet")
		return nil, nil, err
	}
	copy(packet[macSize:], sig)
	// add the hash to the front. Note: this doesn't protect the
	// packet in any way. Our public key will be part of this hash in
	// The future.
	hash = crypto.Keccak256(packet[macSize:])
	copy(packet, hash)
	return packet, hash, nil
}

// readLoop runs in its own goroutine. it handles incoming UDP packets.
func (t *udp) readLoop(unhandled chan<- ReadPacket) {
	defer t.wg.Done()
	if unhandled != nil {
		defer close(unhandled)
	}
	// Discovery packets are defined to be no larger than 1280 bytes.
	// Packets larger than this size will be cut at the end and treated
	// as invalid because their hash won't match.
	buf := make([]byte, 1280)
	for {
		nbytes, from, err := t.conn.ReadFromUDP(buf)
		if netutil.IsTemporaryError(err) {
			// Ignore temporary read errors.
			log.WithError(err).Debug("Temporary UDP read error")
			continue
		} else if err != nil {
			// Shut down the loop for permament errors.
			log.WithError(err).Debug("UDP read error")
			return
		}
		if t.handlePacket(from, buf[:nbytes]) != nil && unhandled != nil {
			select {
			case unhandled <- ReadPacket{buf[:nbytes], from}:
			default:
			}
		}
	}
}

func (t *udp) handlePacket(from *net.UDPAddr, buf []byte) error {
	packet, fromKey, hash, err := decodePacket(buf)
	if err != nil {
		log.WithError(err).WithField("addr", from).Debug("Bad discv4 packet")
		return err
	}
	err = packet.handle(t, from, fromKey, hash)
	log.WithError(err).WithField("from key", fromKey.id().TerminalString()).WithField("addr", from).Trace("<< handle  " + packet.name())
	return err
}

func decodePacket(buf []byte) (packet, EncPubkey, []byte, error) {
	if len(buf) < headSize+1 {
		return nil, EncPubkey{}, nil, errPacketTooSmall
	}
	hash, sig, sigdata := buf[:macSize], buf[macSize:headSize], buf[headSize:]
	shouldhash := crypto.Keccak256(buf[macSize:])
	if !bytes.Equal(hash, shouldhash) {
		return nil, EncPubkey{}, nil, errBadHash
	}
	fromID, err := recoverNodeKey(crypto.Keccak256(buf[headSize:]), sig)
	if err != nil {
		return nil, EncPubkey{}, hash, err
	}
	data := sigdata[1:]
	//var req packet
	switch ptype := sigdata[0]; ptype {
	case pingPacket:
		req := new(Ping)
		_, err = req.UnmarshalMsg(data)
		return req, fromID, hash, err
	case pongPacket:
		req := new(Pong)
		_, err = req.UnmarshalMsg(data)
		return req, fromID, hash, err
	case findnodePacket:
		req := new(Findnode)
		_, err = req.UnmarshalMsg(data)
		return req, fromID, hash, err
	case neighborsPacket:
		req := new(Neighbors)
		_, err = req.UnmarshalMsg(data)
		return req, fromID, hash, err
	default:
		return nil, fromID, hash, fmt.Errorf("unknown type: %d", ptype)
	}
	//s := rlp.NewStream(bytes.NewReader(sigdata[1:]), 0)
	//err = s.Decode(req)

	//return req, fromID, hash, err
	return nil, fromID, hash, err
}

func (req *Ping) handle(t *udp, from *net.UDPAddr, fromKey EncPubkey, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	key, err := decodePubkey(fromKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %v", err)
	}
	pong := &Pong{
		To:         makeEndpoint(from, req.From.TCP),
		ReplyTok:   mac,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	data, _ := pong.MarshalMsg(nil)
	t.send(from, pongPacket, data, pong.name())
	// no need to handle reply ,because we did'n add to pending
	//t.handleReply(fromID, pingPacket,req,from)

	// Add the node to the table. Before doing so, ensure that we have a recent enough pong
	// recorded in the database so their findnode requests will be accepted later.
	n := wrapNode(enode.NewV4(key, from.IP, int(req.From.TCP), from.Port))
	if time.Since(t.db.LastPongReceived(n.ID())) > bondExpiration {
		t.sendPing(n.ID(), from, func() { t.tab.addThroughPing(n) })
	} else {
		t.tab.addThroughPing(n)
	}
	t.localNode.UDPEndpointStatement(from, &net.UDPAddr{IP: req.To.IP, Port: int(req.To.UDP)})
	t.db.UpdateLastPingReceived(n.ID(), time.Now())
	return nil
}

func (req *Ping) name() string { return "PING/v4" }

func (req *Pong) handle(t *udp, from *net.UDPAddr, fromKey EncPubkey, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	fromID := fromKey.id()
	if !t.handleReply(fromID, pongPacket, req) {
		return errUnsolicitedReply
	}
	t.localNode.UDPEndpointStatement(from, &net.UDPAddr{IP: req.To.IP, Port: int(req.To.UDP)})
	t.db.UpdateLastPongReceived(fromID, time.Now())
	return nil
}

func (req *Pong) name() string { return "PONG/v4" }

func (req *Findnode) handle(t *udp, from *net.UDPAddr, fromKey EncPubkey, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	fromID := fromKey.id()
	if time.Since(t.db.LastPongReceived(fromID)) > bondExpiration {
		// No endpoint proof pong exists, we don't process the packet. This prevents an
		// attack vector where the discovery protocol could be used to amplify traffic in a
		// DDOS attack. A malicious actor would send a findnode request with the IP address
		// and UDP port of the target as the source address. The recipient of the findnode
		// packet would then send a neighbors packet (which is a much bigger packet than
		// findnode) to the victim.
		return errUnknownNode
	}
	target := enode.ID(crypto.Keccak256Hash(req.Target[:]).Bytes)
	t.tab.mutex.Lock()
	closest := t.tab.closest(target, bucketSize).entries
	t.tab.mutex.Unlock()

	p := Neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	var sent bool
	// Send neighbors in chunks with at most maxNeighbors per packet
	// to stay below the 1280 byte limit.
	for _, n := range closest {
		if netutil.CheckRelayIP(from.IP, n.IP()) == nil {
			p.Nodes = append(p.Nodes, nodeToRPC(n))
		}
		if len(p.Nodes) == maxNeighbors {
			data, _ := p.MarshalMsg(nil)
			t.send(from, neighborsPacket, data, p.name())
			p.Nodes = p.Nodes[:0]
			sent = true
		}
	}
	if len(p.Nodes) > 0 || !sent {
		data, _ := p.MarshalMsg(nil)
		t.send(from, neighborsPacket, data, p.name())
	}
	return nil
}

func (req *Findnode) name() string { return "FINDNODE/v4" }

func (req *Neighbors) handle(t *udp, from *net.UDPAddr, fromKey EncPubkey, mac []byte) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.handleReply(fromKey.id(), neighborsPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *Neighbors) name() string { return "NEIGHBORS/v4" }

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
