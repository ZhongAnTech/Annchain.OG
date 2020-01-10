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

package common

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp
//msgp:tuple Address

// Length of Addresses in bytes.
const (
	AddressLength = 20
)

var (
	addressT = reflect.TypeOf(Address{})
)

// Address represents the 20 byte of address.
//msgp:tuple Address
type Address struct {
	Bytes [AddressLength]byte `msgp:"bytes"`
}

// BytesToAddress sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToAddress(b []byte) Address {
	var h Address
	h.MustSetBytes(b)
	return h
}

func StringToAddress(s string) (add Address, err error) {
	var h Address
	err = h.SetBytes(FromHex(s))
	return h, err
}

// BigToAddress sets byte representation of b to Address.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress sets byte representation of s to Address.
// If b is larger than len(h), b will be cropped from the left.
func HexToAddress(s string) Address { return BytesToAddress(FromHex(s)) }

// ToBytes convers Address to []byte.
func (h Address) ToBytes() []byte { return h.Bytes[:] }

// Big converts an Address to a big integer.
func (h Address) Big() *big.Int { return new(big.Int).SetBytes(h.Bytes[:]) }

// Hex converts a Address to a hex string.
func (h Address) Hex() string { return hexutil.Encode(h.Bytes[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Address) TerminalString() string {
	return fmt.Sprintf("%x…%x", h.Bytes[:3], h.Bytes[len(h.Bytes)-3:])
}

func (h Address) ShortString() string {
	return hexutil.Encode(h.Bytes[:8])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Address) String() string {
	return h.Hex()
}

// Format implements fmt.Formatter, forcing the byte slice to be formatted as is,
// without going through the stringer interface used for logging.
func (h Address) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h.Bytes)
}

// UnmarshalText parses an Address in hex syntax.
func (h *Address) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Address", input, h.Bytes[:])
}

// UnmarshalJSON parses an Address in hex syntax.
func (h *Address) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(addressT, input, h.Bytes[:])
}

// MarshalText returns the hex representation of h.
func (h Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h.Bytes[:]).MarshalText()
}

// SetBytes sets the Address to the value of b.
// If b is larger than len(h), panic. It usually indicates a logic error.
func (h *Address) MustSetBytes(b []byte) {
	if len(b) > AddressLength {
		panic(fmt.Sprintf("byte to set is longer than expected length: %d > %d", len(b), AddressLength))
	}
	h.Bytes = [AddressLength]byte{}
	copy(h.Bytes[:], b)
}

func (h *Address) SetBytes(b []byte) error {
	if len(b) > AddressLength {
		return fmt.Errorf("byte to set is longer than expected length: %d > %d", len(b), AddressLength)
	}
	h.Bytes = [AddressLength]byte{}
	copy(h.Bytes[:], b)
	return nil
}

// Generate implements testing/quick.Generator.
func (h Address) Generate(rand *rand.Rand, size int) reflect.Value {
	m := rand.Intn(AddressLength)
	for i := AddressLength - 1; i > m; i-- {
		h.Bytes[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(h)
}

func (h Address) EqualTo(b Address) bool {
	return h.Hex() == b.Hex()
}
