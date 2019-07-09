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

package trie

import (
	"hash"
	"sync"

	"github.com/annchain/OG/common"
	"golang.org/x/crypto/sha3"

	log "github.com/sirupsen/logrus"
)

type hasher struct {
	tmp        sliceBuffer
	sha        keccakState
	cachegen   uint16
	cachelimit uint16
	onleaf     LeafCallback
}

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

type sliceBuffer []byte

func (b *sliceBuffer) Write(data []byte) (n int, err error) {
	*b = append(*b, data...)
	return len(data), nil
}

func (b *sliceBuffer) Reset() {
	*b = (*b)[:0]
}

// hashers live in a global db.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{
			tmp: make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
			sha: sha3.NewLegacyKeccak256().(keccakState),
		}
	},
}

func newHasher(cachegen, cachelimit uint16, onleaf LeafCallback) *hasher {
	h := hasherPool.Get().(*hasher)
	h.cachegen, h.cachelimit, h.onleaf = cachegen, cachelimit, onleaf
	return h
}

func returnHasherToPool(h *hasher) {
	hasherPool.Put(h)
}

// hash collapses a node down into a hash node, also returning a copy of the
// original node initialized with the computed hash to replace the original one.
func (h *hasher) hash(n Node, db *Database, force bool) (Node, Node, error) {

	log.Tracef("Panic debug, start hash node: %s", n.String())

	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cache(); hash != nil {
		if db == nil {
			return hash, n, nil
		}
		if n.canUnload(h.cachegen, h.cachelimit) {
			// Unload the node from cache. All of its subnodes will have a lower or equal
			// cache generation number.
			cacheUnloadCounter.Inc(1)
			return hash, hash, nil
		}
		if !dirty {
			return hash, n, nil
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := h.hashChildren(n, db)
	if err != nil {
		return HashNode{}, n, err
	}
	hashed, err := h.store(collapsed, db, force)
	if err != nil {
		return HashNode{}, n, err
	}

	log.Tracef("Panic debug, hashed the node: %s", hashed.String())

	// Cache the hash of the node for later reuse and remove
	// the dirty flag in commit mode. It's fine to assign these values directly
	// without copying the node first because hashChildren copies it.
	cachedHash, _ := hashed.(HashNode)
	switch cn := cached.(type) {
	case *ShortNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	case *FullNode:
		cn.flags.hash = cachedHash
		if db != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}

// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (h *hasher) hashChildren(original Node, db *Database) (Node, Node, error) {
	var err error

	switch n := original.(type) {
	case *ShortNode:
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = common.CopyBytes(n.Key)

		if _, ok := n.Val.(ValueNode); !ok {
			collapsed.Val, cached.Val, err = h.hash(n.Val, db, false)
			if err != nil {
				return original, original, err
			}
		}
		// OG version of trie use msgp encoding, and it can handle nil node.
		// Commend these lines
		// if collapsed.Val == nil {
		// 	collapsed.Val = ValueNode(nil) // Ensure that nil children are encoded as empty strings.
		// }
		return collapsed, cached, nil

	case *FullNode:
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()

		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = h.hash(n.Children[i], db, false)
				if err != nil {
					return original, original, err
				}
			}
			// OG version of trie use msgp encoding, and it can handle nil node.
			// Commend these lines
			// else {
			// 	collapsed.Children[i] = ValueNode(nil) // Ensure that nil children are encoded as empty strings.
			// }
		}
		cached.Children[16] = n.Children[16]
		if collapsed.Children[16] == nil {
			// OG version of trie use msgp encoding, and it can handle nil node.
			// Commend these lines
			// collapsed.Children[16] = ValueNode(nil)
		}
		return collapsed, cached, nil

	default:
		// Value and hash nodes don't have children so they're left as were
		return n, original, nil
	}
}

// store hashes the node n and if we have a storage layer specified, it writes
// the key/value pair to it and tracks any node->child references as well as any
// node->external trie references.
func (h *hasher) store(n Node, db *Database, force bool) (Node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(HashNode); n == nil || isHash {
		return n, nil
	}
	// Generate the msgp encoding of the node
	h.tmp.Reset()
	data := n.encodeNode()
	h.tmp = data
	if len(h.tmp) < 32 && !force {
		return n, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger nodes are replaced by their hash and stored in the database.
	hash, _ := n.cache()
	if hash == nil {
		hash = h.makeHashNode(h.tmp)
	}

	if db != nil {
		// We are pooling the trie nodes into an intermediate memory cache
		db.lock.Lock()
		hash := common.BytesToHash(hash)
		db.insert(hash, h.tmp)
		// Track all direct parent->child node references
		switch n := n.(type) {
		case *ShortNode:
			if child, ok := n.Val.(HashNode); ok {
				db.reference(common.BytesToHash(child), hash)
			}
		case *FullNode:
			for i := 0; i < 16; i++ {
				if child, ok := n.Children[i].(HashNode); ok {
					db.reference(common.BytesToHash(child), hash)
				}
			}
		}
		db.lock.Unlock()

		// Track external references from account->storage trie
		if h.onleaf != nil {
			switch n := n.(type) {
			case *ShortNode:
				if child, ok := n.Val.(ValueNode); ok && child != nil {
					h.onleaf(child, hash)
				}
			case *FullNode:
				for i := 0; i < 16; i++ {
					if child, ok := n.Children[i].(ValueNode); ok && child != nil {
						h.onleaf(child, hash)
					}
				}
			}
		}
	}
	return hash, nil
}

func (h *hasher) makeHashNode(data []byte) HashNode {
	n := make(HashNode, h.sha.Size())
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(n)
	return n
}
