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

// Package enr implements Ethereum Node Records as defined in EIP-778. A node record holds
// arbitrary information about a node on the peer-to-peer network. Node information is
// stored in key/value Pairs. To store and retrieve key/values in a record, use the Entry
// interface.
//
// Signature Handling
//
// Records must be signed before transmitting them to another node.
//
// Decoding a record doesn't check its Signature. Code working with records from an
// untrusted source must always verify two things: that the record uses an identity scheme
// deemed secure, and that the Signature is valid according to the declared scheme.
//
// When creating a record, set the entries you want and use a signing function provided by
// the identity scheme to add the Signature. Modifying a record invalidates the Signature.
//
// Package enr supports the "secp256k1-keccak" identity scheme.
package enr

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/annchain/OG/types/msg"
	"sort"
)

//go:generate msgp
const SizeLimit = 300 // maximum encoded size of a node record in bytes

var (
	ErrInvalidSig     = errors.New("invalid Signature on node record")
	errNotSorted      = errors.New("record key/value Pairs are not sorted by key")
	errDuplicateKey   = errors.New("record contains duplicate key")
	errIncompletePair = errors.New("record contains incomplete k/v Pair")
	errTooBig         = fmt.Errorf("record bigger than %d bytes", SizeLimit)
	errEncodeUnsigned = errors.New("can't encode unsigned record")
	errNotFound       = errors.New("no such key in record")
)

// Pair is a key/value Pair in a record.
//msgp:tuple Pair
type Pair struct {
	K string
	//v rlp.RawValue
	V []byte
}

// Record represents a node record. The zero value is an empty record.
//msgp:tuple Record
type Record struct {
	Seq       uint64 // sequence number
	Signature []byte // the Signature
	Pairs     []Pair // sorted list of all key/value Pairs
	raw       []byte // msgp encoded record
}

// Seq returns the sequence number.
func (r *Record) GetSeq() uint64 {
	return r.Seq
}

// SetSeq updates the record sequence number. This invalidates any Signature on the record.
// Calling SetSeq is usually not required because setting any key in a signed record
// increments the sequence number.
func (r *Record) SetSeq(s uint64) {
	r.Signature = nil
	r.raw = nil
	r.Seq = s
}

// Load retrieves the value of a key/value Pair. The given Entry must be a pointer and will
// be set to the value of the entry in the record.
//
// Errors returned by Load are wrapped in KeyError. You can distinguish decoding errors
// from missing keys using the IsNotFound function.
func (r *Record) Load(e Entry) error {
	i := sort.Search(len(r.Pairs), func(i int) bool { return r.Pairs[i].K >= e.ENRKey() })
	if i < len(r.Pairs) && r.Pairs[i].K == e.ENRKey() {
		if _, err := e.UnmarshalMsg(r.Pairs[i].V); err != nil {
			//if err := rlp.DecodeBytes(r.Pairs[i].V, e); err != nil {
			return &KeyError{Key: e.ENRKey(), Err: err}
		}
		return nil
	}
	return &KeyError{Key: e.ENRKey(), Err: errNotFound}
}

// Set adds or updates the given entry in the record. It panics if the value can't be
// encoded. If the record is signed, Set increments the sequence number and invalidates
// the sequence number.
func (r *Record) Set(e Entry) {
	blob, err := e.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Errorf("enr: can't encode %s: %v", e.ENRKey(), err))
	}
	r.invalidate()

	Pairs := make([]Pair, len(r.Pairs))
	copy(Pairs, r.Pairs)
	i := sort.Search(len(Pairs), func(i int) bool { return Pairs[i].K >= e.ENRKey() })
	switch {
	case i < len(Pairs) && Pairs[i].K == e.ENRKey():
		// element is present at r.Pairs[i]
		Pairs[i].V = blob
	case i < len(r.Pairs):
		// insert Pair before i-th elem
		el := Pair{e.ENRKey(), blob}
		Pairs = append(Pairs, Pair{})
		copy(Pairs[i+1:], Pairs[i:])
		Pairs[i] = el
	default:
		// element should be placed at the end of r.Pairs
		Pairs = append(Pairs, Pair{e.ENRKey(), blob})
	}
	r.Pairs = Pairs
}

func (r *Record) invalidate() {
	if r.Signature != nil {
		r.Seq++
	}
	r.Signature = nil
	r.raw = nil
}

// EncodeRLP implements rlp.Encoder. Encoding fails if
// the record is unsigned.
func (r Record) Encode(b []byte) ([]byte, error) {
	if r.Signature == nil {
		return nil, errEncodeUnsigned
	}
	return r.raw, nil
}

// DecodeRLP implements rlp.Decoder. Decoding verifies the Signature.
func (r *Record) Decode(b []byte) ([]byte, error) {
	dec, err := decodeRecord(b)
	if err != nil {
		return nil, err
	}
	*r = dec
	r.raw = b
	return nil, nil
}

func decodeRecord(b []byte) (dec Record, err error) {
	if len(b) > SizeLimit {
		return dec, errTooBig
	}
	if _, err = dec.UnmarshalMsg(b); err != nil {
		return dec, err
	}
	// The rest of the record contains sorted k/v Pairs.
	var prevkey string
	for i := 0; i < len(dec.Pairs); i++ {
		kv := dec.Pairs[i]
		if i > 0 {
			if kv.K == prevkey {
				return dec, errDuplicateKey
			}
			if kv.K < prevkey {
				return dec, errNotSorted
			}
		}
		prevkey = kv.K
	}
	return dec, err
}

// IdentityScheme returns the name of the identity scheme in the record.
func (r *Record) IdentityScheme() string {
	var id ID
	r.Load(&id)
	return string(id)
}

// VerifySignature checks whether the record is signed using the given identity scheme.
func (r *Record) VerifySignature(s IdentityScheme) error {
	return s.Verify(r, r.Signature)
}

// SetSig sets the record Signature. It returns an error if the encoded record is larger
// than the size limit or if the Signature is invalid according to the passed scheme.
//
// You can also use SetSig to remove the Signature explicitly by passing a nil scheme
// and Signature.
//
// SetSig panics when either the scheme or the Signature (but not both) are nil.
func (r *Record) SetSig(s IdentityScheme, sig []byte) error {
	switch {
	// Prevent storing invalid data.
	case s == nil && sig != nil:
		panic("enr: invalid call to SetSig with non-nil Signature but nil scheme")
	case s != nil && sig == nil:
		panic("enr: invalid call to SetSig with nil Signature but non-nil scheme")
	// Verify if we have a scheme.
	case s != nil:
		if err := s.Verify(r, sig); err != nil {
			return err
		}
		r.Signature = sig
		raw, err := r.MarshalMsg(nil)
		if err == nil {
			if len(raw) > SizeLimit {
				err = errTooBig
			}
		}
		if err != nil {
			r.Signature = nil
			return err
		}
		r.Signature, r.raw = sig, raw
	// Reset otherwise.
	default:
		r.Signature, r.raw = nil, nil
	}
	return nil
}

// AppendElements appends the sequence number and entries to the given slice.
func (r *Record) AppendElements(list []msg.MsgpMember) []msg.MsgpMember {
	seq := msg.Uint64(r.Seq)
	list = append(list, &seq)
	for _, p := range r.Pairs {
		k := msg.String(p.K)
		v := msg.Bytes(p.V)
		list = append(list, &k, &v)
	}
	return list
}

/*
func (r *Record) encode(sig []byte) (raw []byte, err error) {
	list := make([]msg.MsgpMember, 1, 2*len(r.Pairs)+1)
	bs := msg.Bytes(sig)
	list[0] = &bs
	msg := msg.Messages(r.AppendElements(list))
	if raw,err = msg.MarshalMsg(nil);err!=nil{
	//if raw, err = rlp.EncodeToBytes(list); err != nil {
		return nil, err
	}
	if len(raw) > SizeLimit {
		return nil, errTooBig
	}
	return raw, nil
}

*/

func (r Record) Equal(r1 Record) (ok bool, reason error) {
	if r.Seq != r1.Seq {
		return false, fmt.Errorf("seq not equal")
	}
	if len(r.Pairs) != len(r1.Pairs) {
		return false, fmt.Errorf("pairs len not equal")
	}
	for i, p := range r.Pairs {
		if p.K != r1.Pairs[i].K {
			return false, fmt.Errorf("pair key is diff")
		}
		if !bytes.Equal(p.V, r.Pairs[i].V) {
			return false, fmt.Errorf("pair val is diff")
		}
	}

	if !bytes.Equal(r.Signature, r1.Signature) {
		return false, fmt.Errorf("signatrue not equal")
	}
	if !bytes.Equal(r.raw, r1.raw) {
		return false, fmt.Errorf("raw not equal")
	}
	return true, nil
}
