// Copyright 2014 The go-ethereum Authors
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
package p2p

import (
	"fmt"
	"github.com/annchain/OG/p2p/enode"
	"github.com/annchain/OG/p2p/enr"
)

type MsgCodeType uint16

// Protocol represents a P2P subprotocol implementation.
type Protocol struct {
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string

	// Version should contain the version number of the protocol.
	Version uint

	// Length should contain the number of message codes used
	// by the protocol.
	Length MsgCodeType

	// Run is called in a new groutine when the protocol has been
	// negotiated with a peer. It should read and write messages from
	// rw. The Payload for each message must be fully consumed.
	//
	// The peer connection is closed when Start returns. It should return
	// any protocol-level error (suc01h as an I/O error) that is
	// encountered.
	Run func(peer *Peer, rw MsgReadWriter) error

	// NodeInfo is an optional helper method to retrieve protocol specific metadata
	// about the host node.
	NodeInfo func() interface{}

	// PeerInfo is an optional helper method to retrieve protocol specific metadata
	// about a certain peer in the network. If an info retrieval function is set,
	// but returns nil, it is assumed that the protocol handshake is still running.
	PeerInfo func(id enode.ID) interface{}

	// Attributes contains protocol specific information for the node record.
	Attributes []enr.Entry
}

func (p Protocol) cap() Cap {
	return Cap{p.Name, p.Version}
}

// Cap is the structure of a peer capability.
type Cap struct {
	Name    string
	Version uint
}

// protoHandshake is the RLP structure of the protocol handshake.
type ProtoHandshake struct {
	Version    uint64 `msg:"version"`
	Name       string `msg:"name"`
	Caps       []Cap  `msg:"caps"`
	ListenPort uint64 `msg:"listen_port"`
	ID         []byte `msg:"id"` // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest [][]byte `rlp:"tail" msg:"tail"`
}

func (cap Cap) RlpData() interface{} {
	return []interface{}{cap.Name, cap.Version}
}

func (cap Cap) String() string {
	return fmt.Sprintf("%s/%d", cap.Name, cap.Version)
}

type CapsByNameAndVersion []Cap

func (cs CapsByNameAndVersion) Len() int      { return len(cs) }
func (cs CapsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs CapsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}

func (CapsByNameAndVersion) ENRKey() string { return "cap" }
