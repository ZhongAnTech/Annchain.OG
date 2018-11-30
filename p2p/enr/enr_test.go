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

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common/msg"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomString(strlen int) string {
	b := make([]byte, strlen)
	rnd.Read(b)
	return string(b)
}

// TestGetSetID tests encoding/decoding and setting/getting of the ID key.
func TestGetSetID(t *testing.T) {
	id := ID("someid")
	var r Record
	r.Set(&id)

	var id2 ID
	require.NoError(t, r.Load(&id2))
	assert.Equal(t, id, id2)
}

// TestGetSetIP4 tests encoding/decoding and setting/getting of the IP key.
func TestGetSetIP4(t *testing.T) {
	ip := IP{192, 168, 0, 3}
	var r Record
	r.Set(&ip)

	var ip2 IP
	require.NoError(t, r.Load(&ip2))
	assert.Equal(t, ip, ip2)
}

// TestGetSetIP6 tests encoding/decoding and setting/getting of the IP key.
func TestGetSetIP6(t *testing.T) {
	ip := IP{0x20, 0x01, 0x48, 0x60, 0, 0, 0x20, 0x01, 0, 0, 0, 0, 0, 0, 0x00, 0x68}
	var r Record
	r.Set(&ip)

	var ip2 IP
	require.NoError(t, r.Load(&ip2))
	assert.Equal(t, ip, ip2)
}

// TestGetSetDiscPort tests encoding/decoding and setting/getting of the DiscPort key.
func TestGetSetUDP(t *testing.T) {
	port := UDP(30309)
	var r Record
	r.Set(&port)

	var port2 UDP
	require.NoError(t, r.Load(&port2))
	assert.Equal(t, port, port2)
}

func TestLoadErrors(t *testing.T) {
	var r Record
	ip4 := IP{127, 0, 0, 1}
	r.Set(&ip4)

	// Check error for missing keys.
	var udp UDP
	err := r.Load(&udp)
	if !IsNotFound(err) {
		t.Error("IsNotFound should return true for missing key")
	}
	assert.Equal(t, &KeyError{Key: udp.ENRKey(), Err: errNotFound}, err)

	// Check error for invalid keys.
	var list msg.Uints
	err = r.Load(WithEntry(ip4.ENRKey(), &list))
	kerr, ok := err.(*KeyError)
	if !ok {
		t.Fatalf("expected KeyError, got %T", err)
	}
	assert.Equal(t, kerr.Key, ip4.ENRKey())
	assert.Error(t, kerr.Err)
	if IsNotFound(err) {
		t.Error("IsNotFound should return false for decoding errors")
	}
}

// TestSortedGetAndSet tests that Set produced a sorted Pairs slice.
func TestSortedGetAndSet(t *testing.T) {
	type Pair struct {
		K string
		V uint32
	}

	for _, tt := range []struct {
		input []Pair
		want  []Pair
	}{
		{
			input: []Pair{{"a", 1}, {"c", 2}, {"b", 3}},
			want:  []Pair{{"a", 1}, {"b", 3}, {"c", 2}},
		},
		{
			input: []Pair{{"a", 1}, {"c", 2}, {"b", 3}, {"d", 4}, {"a", 5}, {"bb", 6}},
			want:  []Pair{{"a", 5}, {"b", 3}, {"bb", 6}, {"c", 2}, {"d", 4}},
		},
		{
			input: []Pair{{"c", 2}, {"b", 3}, {"d", 4}, {"a", 5}, {"bb", 6}},
			want:  []Pair{{"a", 5}, {"b", 3}, {"bb", 6}, {"c", 2}, {"d", 4}},
		},
	} {
		var r Record
		for _, i := range tt.input {
			val := msg.Uint(i.V)
			r.Set(WithEntry(i.K, &val))
		}
		for i, w := range tt.want {
			// set got's key from r.Pair[i], so that we preserve order of Pairs
			got := Pair{K: r.Pairs[i].K}
			val := msg.Uint(got.V)
			assert.NoError(t, r.Load(WithEntry(w.K, &val)))
			got.V = uint32(val)
			assert.Equal(t, w, got)
		}
	}
}

// TestDirty tests record Signature removal on setting of new key/value Pair in record.
func TestDirty(t *testing.T) {
	var r Record

	if _, err := r.Encode(nil); err != errEncodeUnsigned {
		t.Errorf("expected errEncodeUnsigned, got %#v", err)
	}

	require.NoError(t, signTest([]byte{5}, &r))
	if len(r.Signature) == 0 {
		t.Error("record is not signed")
	}
	_, err := r.Encode(nil)
	assert.NoError(t, err)

	r.SetSeq(3)
	if len(r.Signature) != 0 {
		t.Error("Signature still set after modification")
	}
	if _, err := r.Encode(nil); err != errEncodeUnsigned {
		t.Errorf("expected errEncodeUnsigned, got %#v", err)
	}
}

func TestSeq(t *testing.T) {
	var r Record

	assert.Equal(t, uint64(0), r.GetSeq())
	u := UDP(1)
	r.Set(&u)
	assert.Equal(t, uint64(0), r.GetSeq())
	signTest([]byte{5}, &r)
	assert.Equal(t, uint64(0), r.GetSeq())
	u2 := UDP(2)
	r.Set(&u2)
	assert.Equal(t, uint64(1), r.GetSeq())
}

// TestGetSetOverwrite tests value overwrite when setting a new value with an existing key in record.
func TestGetSetOverwrite(t *testing.T) {
	var r Record

	ip := IP{192, 168, 0, 3}
	r.Set(&ip)

	ip2 := IP{192, 168, 0, 4}
	r.Set(&ip2)

	var ip3 IP
	require.NoError(t, r.Load(&ip3))
	assert.Equal(t, ip2, ip3)
}

// TestSignEncodeAndDecode tests signing, RLP encoding and RLP decoding of a record.
func TestSignEncodeAndDecode(t *testing.T) {
	var r Record
	u := UDP(30303)
	r.Set(&u)
	ip := IP{127, 0, 0, 1}
	r.Set(&ip)
	require.NoError(t, signTest([]byte{5}, &r))

	blob, err := r.Encode(nil)
	require.NoError(t, err)

	var r2 Record
	_, err = r2.Decode(blob)
	require.NoError(t, err)
	assert.Equal(t, r, r2)

	blob2, err := r2.Encode(nil)
	require.NoError(t, err)
	assert.Equal(t, blob, blob2)
}

// TestRecordTooBig tests that records bigger than SizeLimit bytes cannot be signed.
func TestRecordTooBig(t *testing.T) {
	var r Record
	key := randomString(10)

	// set a big value for random key, expect error
	str := msg.String(randomString(SizeLimit))
	r.Set(WithEntry(key, &str))
	if err := signTest([]byte{5}, &r); err != errTooBig {
		t.Fatalf("expected to get errTooBig, got %#v", err)
	}
	str2 := msg.String(randomString(100))
	// set an acceptable value for random key, expect no error
	r.Set(WithEntry(key, &str2))
	require.NoError(t, signTest([]byte{5}, &r))
}

// TestSignEncodeAndDecodeRandom tests encoding/decoding of records containing random key/value Pairs.
func TestSignEncodeAndDecodeRandom(t *testing.T) {
	var r Record

	// random key/value Pairs for testing
	Pairs := map[string]uint32{}
	for i := 0; i < 10; i++ {
		key := randomString(7)
		value := rnd.Uint32()
		Pairs[string(key)] = value
		v := msg.Uint(value)
		r.Set(WithEntry(string(key), &v))
	}

	require.NoError(t, signTest([]byte{5}, &r))
	_, err := r.MarshalMsg(nil)
	require.NoError(t, err)

	for k, v := range Pairs {
		desc := fmt.Sprintf("key %q", k)
		var got msg.Uint32
		buf := WithEntry(k, &got)
		require.NoError(t, r.Load(buf), desc)
		require.Equal(t, v, uint32(got), desc)
	}
}

type testSig struct{}

type testID struct {
	msg.Bytes
}

func newTestId(b []byte) *testID {
	return &testID{b}
}

func (id testID) ENRKey() string { return "testid" }

func signTest(id []byte, r *Record) error {
	i := ID("test")
	r.Set(&i)
	v := newTestId(id)
	r.Set(v)
	return r.SetSig(testSig{}, makeTestSig(id, r.GetSeq()))
}

func makeTestSig(id []byte, seq uint64) []byte {
	sig := make([]byte, 8, len(id)+8)
	binary.BigEndian.PutUint64(sig[:8], seq)
	sig = append(sig, id...)
	return sig
}

func (testSig) Verify(r *Record, sig []byte) error {
	var id testID
	if err := r.Load(&id); err != nil {
		return err
	}
	if !bytes.Equal(sig, makeTestSig(id.Bytes, r.GetSeq())) {
		return ErrInvalidSig
	}
	return nil
}

func (testSig) NodeAddr(r *Record) []byte {
	var id testID
	if err := r.Load(&id); err != nil {
		return nil
	}
	return id.Bytes
}
