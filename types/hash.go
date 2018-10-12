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

package types

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	"bytes"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"strings"
)

//go:generate msgp
//msgp:tuple Hash

// Length of hash in bytes.
const (
	HashLength = 32
)

var (
	hashT = reflect.TypeOf(Hash{})
)

// Hash represents the 20 byte of Hash.
type Hash struct {
	Bytes [HashLength]byte `msgp:"bytes"`
}

type Hashs []Hash

type HashBytes [HashLength]byte

func (h *Hash) Empty() bool {
	var hs Hash
	return bytes.Equal(h.Bytes[:], hs.Bytes[:])
}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.MustSetBytes(b)
	return h
}

// BigToHash sets byte representation of b to Hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to Hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(common.FromHex(s)) }

func HexStringToHash(s string) (Hash, error) {
	var h Hash
	err := h.SetBytes(common.FromHex(s))
	return h, err
}

func HashesToString(hashes []Hash) string {
	var strs []string
	for _, v := range hashes {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

// ToBytes convers hash to []byte.
func (h Hash) ToBytes() []byte { return h.Bytes[:] }

// Big converts an Hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h.Bytes[:]) }

// Hex converts a Hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h.Bytes[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h.Bytes[:3], h.Bytes[len(h.Bytes)-3:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()[:10]
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
//func (h Hash) Format(s fmt.State, c rune) {
//	fmt.Fprintf(s, "%"+string(c), h.Bytes)
//}

// UnmarshalText parses an Hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h.Bytes[:])
}

// UnmarshalJSON parses an Hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h.Bytes[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h.Bytes[:]).MarshalText()
}

// SetBytes sets the Hash to the value of b.
// If b is larger than len(h), panic. It usually indicates a logic error.
func (h *Hash) MustSetBytes(b []byte) {
	if len(b) > HashLength {
		panic(fmt.Sprintf("byte to set is longer than expected length: %d > %d", len(b), HashLength))
	}
	h.Bytes = [HashLength]byte{}
	copy(h.Bytes[:], b)
}

func (h *Hash) SetBytes(b []byte) error {
	if len(b) > HashLength {
		return fmt.Errorf("byte to set is longer than expected length: %d > %d", len(b), HashLength)
	}
	h.Bytes = [HashLength]byte{}
	copy(h.Bytes[:], b)
	return nil
}

// Generate implements testing/quick.Generator.
func (h Hash) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(HashLength)
	for i := HashLength - 1; i > m; i-- {
		h.Bytes[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

// Cmp compares two hashes.
// Returns  0 if two hashes are same.
// Returns -1 if the self hash is less than parameter hash.
// Returns  1 if the self hash is larger than parameter hash.
func (h Hash) Cmp(another Hash) int {
	for i := 0; i < HashLength; i++ {
		if h.Bytes[i] < another.Bytes[i] {
			return -1
		} else if h.Bytes[i] > another.Bytes[i] {
			return 1
		}
	}
	return 0
}
