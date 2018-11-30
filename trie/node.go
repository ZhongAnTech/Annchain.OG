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

	log "github.com/sirupsen/logrus"
	"github.com/tinylib/msgp/msgp"
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

type Node interface {
	fstring(string) string
	nodeType() nodetype
	cache() (HashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
	encodeNode() []byte
	DecodeMsg(dc *msgp.Reader) error
	EncodeMsg(en *msgp.Writer) error
	MarshalMsg(b []byte) ([]byte, error)
	UnmarshalMsg(bts []byte) ([]byte, error)
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

// EncodeRLP encodes a full node into the consensus RLP format.
// func (n *fullNode) EncodeRLP(w io.Writer) error {
// 	return rlp.Encode(w, n.Children)
// }

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
	log.Debugf("encode FullNode")
	data, _ := n.MarshalMsg(nil)
	log.Debugf("after encode FullNode, get byte: %x", data)
	return append(encodePrefixFullNode, data...)
}
func (n *ShortNode) encodeNode() []byte {
	log.Debugf("encode ShortNode, key: %x", n.Key)
	data, _ := n.MarshalMsg(nil)
	log.Debugf("after encode ShortNode, get byte: %x", data)
	return append(encodePrefixShortNode, data...)
}
func (n HashNode) encodeNode() []byte {
	log.Debugf("encode HashNode, node: %x", []byte(n))
	data, _ := n.MarshalMsg(nil)
	log.Debugf("after encode HashNode, get byte: %x", data)
	return append(encodePrefixHashNode, data...)
}
func (n ValueNode) encodeNode() []byte {
	log.Debugf("encode ValueNode, node: %x", []byte(n))
	data, _ := n.MarshalMsg(nil)
	log.Debugf("after encode ValueNode, get byte: %x", data)
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
	log.Debugf("decode node hash: %x, data: %x", hash, buf)
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	prefix := buf[:len(encodePrefixFullNode)]
	data := buf[len(encodePrefixFullNode):]

	if bytes.Equal(prefix, encodePrefixFullNode) {
		n, err := decodeFull(hash, data, cachegen)
		log.Debugf("decode FullNode hash: %x", hash)
		return n, wrapError(err, "full")
	}
	if bytes.Equal(prefix, encodePrefixShortNode) {
		n, err := decodeShort(hash, data, cachegen)
		log.Debugf("decode ShortNode hash: %x", hash)
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

	// n := &fullNode{flags: nodeFlag{hash: hash, gen: cachegen}}
	// for i := 0; i < 16; i++ {
	// 	cld, rest, err := decodeRef(elems, cachegen)
	// 	if err != nil {
	// 		return n, wrapError(err, fmt.Sprintf("[%d]", i))
	// 	}
	// 	n.Children[i], elems = cld, rest
	// }
	// val, _, err := rlp.SplitString(elems)
	// if err != nil {
	// 	return n, err
	// }
	// if len(val) > 0 {
	// 	n.Children[16] = append(valueNode{}, val...)
	// }
	// return n, nil
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

	// kbuf, rest, err := rlp.SplitString(elems)
	// if err != nil {
	// 	return nil, err
	// }
	// flag := nodeFlag{hash: hash, gen: cachegen}
	// key := compactToHex(kbuf)
	// if hasTerm(key) {
	// 	// value node
	// 	val, _, err := rlp.SplitString(rest)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("invalid value node: %v", err)
	// 	}
	// 	return &shortNode{key, append(valueNode{}, val...), flag}, nil
	// }
	// r, _, err := decodeRef(rest, cachegen)
	// if err != nil {
	// 	return nil, wrapError(err, "val")
	// }
	// return &shortNode{key, r, flag}, nil
}

// const hashLen = len(types.Hash{})

// func decodeRef(buf []byte, cachegen uint16) (node, []byte, error) {
// 	kind, val, rest, err := rlp.Split(buf)
// 	if err != nil {
// 		return nil, buf, err
// 	}
// 	switch {
// 	case kind == rlp.List:
// 		// 'embedded' node reference. The encoding must be smaller
// 		// than a hash in order to be valid.
// 		if size := len(buf) - len(rest); size > hashLen {
// 			err := fmt.Errorf("oversized embedded node (size is %d bytes, want size < %d)", size, hashLen)
// 			return nil, buf, err
// 		}
// 		n, err := decodeNode(nil, buf, cachegen)
// 		return n, rest, err
// 	case kind == rlp.String && len(val) == 0:
// 		// empty node
// 		return nil, rest, nil
// 	case kind == rlp.String && len(val) == 32:
// 		return append(hashNode{}, val...), rest, nil
// 	default:
// 		return nil, nil, fmt.Errorf("invalid RLP string size %d (want 0 or 32)", len(val))
// 	}
// }

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
