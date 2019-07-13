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

package trie

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/tinylib/msgp/msgp"
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

type Node interface {
	fstring(string) string
	nodeType() nodetype
	cache() (HashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
	encodeNode() []byte
	String() string
	DecodeMsg(dc *msgp.Reader) error
	EncodeMsg(en *msgp.Writer) error
	MarshalMsg(b []byte) ([]byte, error)
	Msgsize() int
}

//go:generate msgp

//msgp:tuple FullNode
type FullNode struct {
	Children [17]Node // Actual trie node data to encode/decode (needs custom encoder)
	flags    nodeFlag
}

//msgp:tuple ShortNode
type ShortNode struct {
	Key   []byte
	Val   Node
	flags nodeFlag
}

//msgp: HashNode
type HashNode []byte

//msgp: ValueNode
type ValueNode []byte

func (n *FullNode) copy() *FullNode   { copy := *n; return &copy }
func (n *ShortNode) copy() *ShortNode { copy := *n; return &copy }

// nodeFlag contains caching-related metadata about a node.
type nodeFlag struct {
	hash  HashNode // cached hash of the node (may be nil)
	gen   uint16   // cache generation counter
	dirty bool     // whether the node has changes that must be written to the database
}

// canUnload tells whether a node can be unloaded.
func (n *nodeFlag) canUnload(cachegen, cachelimit uint16) bool {
	return !n.dirty && cachegen-n.gen >= cachelimit
}

type nodetype int

const (
	nilnode nodetype = iota
	fullnode
	shortnode
	hashnode
	valuenode
)

func (n *FullNode) nodeType() nodetype  { return fullnode }
func (n *ShortNode) nodeType() nodetype { return shortnode }
func (n HashNode) nodeType() nodetype   { return hashnode }
func (n ValueNode) nodeType() nodetype  { return valuenode }

func (n *FullNode) canUnload(gen, limit uint16) bool  { return n.flags.canUnload(gen, limit) }
func (n *ShortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n HashNode) canUnload(uint16, uint16) bool      { return false }
func (n ValueNode) canUnload(uint16, uint16) bool     { return false }

func (n *FullNode) cache() (HashNode, bool)  { return n.flags.hash, n.flags.dirty }
func (n *ShortNode) cache() (HashNode, bool) { return n.flags.hash, n.flags.dirty }
func (n HashNode) cache() (HashNode, bool)   { return nil, true }
func (n ValueNode) cache() (HashNode, bool)  { return nil, true }

// Pretty printing.
func (n *FullNode) String() string  { return n.fstring("") }
func (n *ShortNode) String() string { return n.fstring("") }
func (n HashNode) String() string   { return n.fstring("") }
func (n ValueNode) String() string  { return n.fstring("") }

func (n *FullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range n.Children {
		if node == nil {
			resp += fmt.Sprintf("%s: <nil> ", indices[i])
		} else {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}
	return resp + fmt.Sprintf("\n%s] ", ind)
}
func (n *ShortNode) fstring(ind string) string {
	return fmt.Sprintf("{%x: %v} ", n.Key, n.Val.fstring(ind+"  "))
}
func (n HashNode) fstring(ind string) string {
	return fmt.Sprintf("<%x> ", []byte(n))
}
func (n ValueNode) fstring(ind string) string {
	return fmt.Sprintf("%x ", []byte(n))
}

var (
	encodePrefixFullNode  = []byte("f")
	encodePrefixShortNode = []byte("s")
	encodePrefixHashNode  = []byte("h")
	encodePrefixValueNode = []byte("v")
)

func (n *FullNode) encodeNode() []byte {
	data, _ := n.MarshalMsg(nil)
	return append(encodePrefixFullNode, data...)
}
func (n *ShortNode) encodeNode() []byte {
	data, _ := n.MarshalMsg(nil)
	return append(encodePrefixShortNode, data...)
}
func (n HashNode) encodeNode() []byte {
	data, _ := n.MarshalMsg(nil)
	return append(encodePrefixHashNode, data...)
}
func (n ValueNode) encodeNode() []byte {
	data, _ := n.MarshalMsg(nil)
	return append(encodePrefixValueNode, data...)
}

func mustDecodeNode(hash, buf []byte, cachegen uint16) Node {
	n, err := decodeNode(hash, buf, cachegen)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", hash, err))
	}
	return n
}

// decodeNode parses the msgp encoding of a trie node.
func decodeNode(hash, buf []byte, cachegen uint16) (Node, error) {
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	prefix := buf[:len(encodePrefixFullNode)]
	data := buf[len(encodePrefixFullNode):]

	if bytes.Equal(prefix, encodePrefixFullNode) {
		n, err := decodeFull(hash, data, cachegen)

		//log.Tracef("Panic debug, decode Fullnode, hash: %x, data: %x, fullnode: %s", hash, data, n.String())
		return n, wrapError(err, "full")
	}
	if bytes.Equal(prefix, encodePrefixShortNode) {
		n, err := decodeShort(hash, data, cachegen)

		//log.Tracef("Panic debug, decode Shortnode, hash: %x, data: %x, shortnode: %s", hash, data, n.String())
		return n, wrapError(err, "short")
	}
	return nil, fmt.Errorf("invalid prefix of encoded node: %v", prefix)
}

func decodeFull(hash, data []byte, cachegen uint16) (*FullNode, error) {
	var n FullNode
	_, err := n.UnmarshalMsg(data)
	if err != nil {
		return &n, err
	}

	flag := nodeFlag{hash: hash, gen: cachegen}
	n.flags = flag
	return &n, nil
}

func decodeShort(hash, data []byte, cachegen uint16) (Node, error) {
	var n ShortNode
	_, err := n.UnmarshalMsg(data)
	if err != nil {
		return &n, err
	}
	n.Key = compactToHex(n.Key)

	flag := nodeFlag{hash: hash, gen: cachegen}
	n.flags = flag

	return &n, nil
}

// wraps a decoding error with information about the path to the
// invalid child node (for debugging encoding issues).
type decodeError struct {
	what  error
	stack []string
}

func wrapError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	if decErr, ok := err.(*decodeError); ok {
		decErr.stack = append(decErr.stack, ctx)
		return decErr
	}
	return &decodeError{err, []string{ctx}}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("%v (decode path: %s)", err.what, strings.Join(err.stack, "<-"))
}
