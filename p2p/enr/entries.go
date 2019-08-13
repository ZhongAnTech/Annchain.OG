// Copyright 2017 The go-ethereum Authors
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

package enr

//go:generate msgp

import (
	"fmt"
	"github.com/annchain/OG/types/msg"
	"github.com/tinylib/msgp/msgp"
	"net"
)

// Entry is implemented by known node record entry types.
//
// To define a new entry that is to be included in a node record,
// create a Go type that satisfies this interface. The type should
// also implement rlp.Decoder if additional checks are needed on the value.
type Entry interface {
	ENRKey() string
	msg.MsgpMember
}

type generic struct {
	key string
	//value interface{}
	value msg.MsgpMember
}

func (g generic) ENRKey() string { return g.key }

func (g generic) MarshalMsg(b []byte) ([]byte, error) {
	return g.value.MarshalMsg(b)
}

func (g *generic) UnmarshalMsg(b []byte) ([]byte, error) {
	return g.value.UnmarshalMsg(b)
}

func (g *generic) Msgsize() int {
	return g.value.Msgsize()
}

func (g *generic) DecodeMsg(en *msgp.Reader) (err error) {
	return g.value.DecodeMsg(en)
}

func (g generic) EncodeMsg(en *msgp.Writer) (err error) {
	return g.value.EncodeMsg(en)
}

// WithEntry wraps any value with a key name. It can be used to set and load arbitrary values
// in a record. The value v must be supported by rlp. To use WithEntry with Load, the value
// must be a pointer.
func WithEntry(k string, v msg.MsgpMember) Entry {
	return &generic{key: k, value: v}
}

// TCP is the "tcp" key, which holds the TCP port of the node.
type TCP uint16

func (v TCP) ENRKey() string { return "tcp" }

// UDP is the "udp" key, which holds the UDP port of the node.
type UDP uint16

func (v UDP) ENRKey() string { return "udp" }

// ID is the "id" key, which holds the name of the identity scheme.
type ID string

const IDv4 = ID("v4") // the default identity scheme

func (v ID) ENRKey() string { return "id" }

// IP is the "ip" key, which holds the IP address of the node.
type IP net.IP

func (v IP) ENRKey() string { return "ip" }

// EncodeRLP implements rlp.Encoder.
func (v IP) MarshalMsg(b []byte) ([]byte, error) {
	if ip4 := net.IP(v).To4(); ip4 != nil {
		bs := msg.Bytes(ip4)
		return bs.MarshalMsg(b)
	}
	bs := msg.Bytes(v)
	return bs.MarshalMsg(b)
}

func (v IP) Msgsize() int {
	bs := msg.Bytes(v)
	return bs.Msgsize()
}

// DecodeRLP implements rlp.Decoder.
func (v *IP) UnmarshalMsg(b []byte) ([]byte, error) {
	var bs msg.Bytes
	d, err := bs.UnmarshalMsg(b)
	if err != nil {
		return d, err
	}
	if len(bs) != 4 && len(bs) != 16 {
		return d, fmt.Errorf("invalid IP address, want 4 or 16 bytes: %v", *v)
	}
	*v = IP(bs)
	return d, nil
}

func (v *IP) DecodeMsg(en *msgp.Reader) error {
	var bs msg.Bytes
	err := bs.DecodeMsg(en)
	if err != nil {
		return err
	}
	if len(bs) != 4 && len(bs) != 16 {
		return fmt.Errorf("invalid IP address, want 4 or 16 bytes: %v", *v)
	}
	*v = IP(bs)
	return nil
}

func (v IP) EncodeMsg(en *msgp.Writer) (err error) {
	if ip4 := net.IP(v).To4(); ip4 != nil {
		bs := msg.Bytes(ip4)
		return bs.EncodeMsg(en)
	}
	bs := msg.Bytes(v)
	return bs.EncodeMsg(en)
}
