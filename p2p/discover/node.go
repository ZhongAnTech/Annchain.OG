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
	"errors"
	crypto2 "github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/p2p/onode"
	"math/big"
	"net"
	"time"
)

//go:generate msgp
// node represents a host on the network.
// The fields of Node may not be modified.
type node struct {
	onode.Node
	addedAt time.Time // time when the node was added to the table
}

type EncPubkey [64]byte

func encodePubkey(key *ecdsa.PublicKey) EncPubkey {
	var e EncPubkey
	math.ReadBits(key.X, e[:len(e)/2])
	math.ReadBits(key.Y, e[len(e)/2:])
	return e
}

func decodePubkey(e EncPubkey) (*ecdsa.PublicKey, error) {
	p := &ecdsa.PublicKey{Curve: crypto2.S256(), X: new(big.Int), Y: new(big.Int)}
	half := len(e) / 2
	p.X.SetBytes(e[:half])
	p.Y.SetBytes(e[half:])
	if !p.Curve.IsOnCurve(p.X, p.Y) {
		return nil, errors.New("invalid secp256k1 curve point")
	}
	return p, nil
}

func (e EncPubkey) id() onode.ID {
	return onode.ID(crypto2.Keccak256Hash(e[:]).Bytes)
}

// recoverNodeKey computes the public key used to sign the
// given hash from the signature.
func recoverNodeKey(hash, sig []byte) (key EncPubkey, err error) {
	pubkey, err := crypto2.Ecrecover(hash, sig)
	if err != nil {
		return key, err
	}
	copy(key[:], pubkey[1:])
	return key, nil
}

func wrapNode(n *onode.Node) *node {
	return &node{Node: *n}
}

func wrapNodes(ns []*onode.Node) []*node {
	result := make([]*node, len(ns))
	for i, n := range ns {
		result[i] = wrapNode(n)
	}
	return result
}

func unwrapNode(n *node) *onode.Node {
	return &n.Node
}

func unwrapNodes(ns []*node) []*onode.Node {
	result := make([]*onode.Node, len(ns))
	for i, n := range ns {
		result[i] = unwrapNode(n)
	}
	return result
}

func (n *node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP(), Port: n.UDP()}
}

func (n *node) String() string {
	return n.Node.String()
}
