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

package discover

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"sync"

	"github.com/annchain/OG/p2p/onode"
	"github.com/annchain/OG/p2p/enr"
)

var nullNode *onode.Node

func init() {
	var r enr.Record
	ip := enr.IP{0, 0, 0, 0}
	r.Set(&ip)
	nullNode = onode.SignNull(&r, onode.ID{})
}

func newTestTable(t transport) (*Table, *onode.DB) {
	db, _ := onode.OpenDB("")
	tab, _ := newTable(t, db, nil)
	return tab, db
}

// nodeAtDistance creates a node for which onode.LogDist(base, n.id) == ld.
func nodeAtDistance(base onode.ID, ld int, ip net.IP) *node {
	var r enr.Record
	eIP := enr.IP(ip)
	r.Set(&eIP)
	return wrapNode(onode.SignNull(&r, idAtDistance(base, ld)))
}

// idAtDistance returns a random hash such that onode.LogDist(a, b) == n
func idAtDistance(a onode.ID, n int) (b onode.ID) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func intIP(i int) net.IP {
	return net.IP{byte(i), 0, 2, byte(i)}
}

// fillBucket inserts nodes into the given bucket until it is full.
func fillBucket(tab *Table, n *node) (last *node) {
	ld := onode.LogDist(tab.self().ID(), n.ID())
	b := tab.bucket(n.ID())
	for len(b.entries) < bucketSize {
		b.entries = append(b.entries, nodeAtDistance(tab.self().ID(), ld, intIP(ld)))
	}
	return b.entries[bucketSize-1]
}

type pingRecorder struct {
	mu           sync.Mutex
	dead, pinged map[onode.ID]bool
	n            *onode.Node
}

func newPingRecorder() *pingRecorder {
	var r enr.Record
	eIp := enr.IP{0, 0, 0, 0}
	r.Set(&eIp)
	n := onode.SignNull(&r, onode.ID{})

	return &pingRecorder{
		dead:   make(map[onode.ID]bool),
		pinged: make(map[onode.ID]bool),
		n:      n,
	}
}

func (t *pingRecorder) self() *onode.Node {
	return nullNode
}

func (t *pingRecorder) findnode(toid onode.ID, toaddr *net.UDPAddr, target EncPubkey) ([]*node, error) {
	return nil, nil
}

func (t *pingRecorder) waitping(from onode.ID) error {
	return nil // remote always pings
}

func (t *pingRecorder) ping(toid onode.ID, toaddr *net.UDPAddr) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pinged[toid] = true
	if t.dead[toid] {
		return errTimeout
	} else {
		return nil
	}
}

func (t *pingRecorder) close() {}

func hasDuplicates(slice []*node) bool {
	seen := make(map[onode.ID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID()] {
			return true
		}
		seen[e.ID()] = true
	}
	return false
}

func contains(ns []*node, id onode.ID) bool {
	for _, n := range ns {
		if n.ID() == id {
			return true
		}
	}
	return false
}

func sortedByDistanceTo(distbase onode.ID, slice []*node) bool {
	var last onode.ID
	for i, e := range slice {
		if i > 0 && onode.DistCmp(distbase, e.ID(), last) < 0 {
			return false
		}
		last = e.ID()
	}
	return true
}

func hexEncPubkey(h string) (ret EncPubkey) {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	if len(b) != len(ret) {
		panic("invalid length")
	}
	copy(ret[:], b)
	return ret
}

func hexPubkey(h string) *ecdsa.PublicKey {
	k, err := decodePubkey(hexEncPubkey(h))
	if err != nil {
		panic(err)
	}
	return k
}
