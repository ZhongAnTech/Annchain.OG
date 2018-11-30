// Copyright 2018 The go-ethereum Authors
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

package enode

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/annchain/OG/p2p/enr"
	"github.com/stretchr/testify/assert"
)

//var pyRecord, _ = hex.DecodeString("f884b8407098ad865b00a582051940cb9cf36836572411a47278783077011599ed5cd16b76f2635f4e234738f30813a89eb9137e3e3df5266e3a1f11df72ecf1145ccb9c01826964827634826970847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31388375647082765f")
var pyRecord, _ = hex.DecodeString("83a353657101a95369676e6174757265c4401fb4a09b1eb072d998b35ca3c681902d3866507c527eb612fffbb021cbd110c34087a517948b6130914fca1cb7acb52e1c4ea4779cfb93cb74de731bf7d2415fa550616972739482a14ba26964a156c403a2763482a14ba26970a156c406c4047f00000182a14ba9736563703235366b31a156c423c42103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd313882a14ba3756470a156c403cd765f")

// TestPythonInterop checks that we can decode and verify a record produced by the Python
// implementation.
func TestPythonInterop(t *testing.T) {
	//var r1 enr.Record
	var (
		wantID  = HexID("a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7")
		wantSeq = uint64(1)
		wantIP  = enr.IP{127, 0, 0, 1}
		wantUDP = enr.UDP(30303)
	)
	//r1.Set(&wantIP)
	//r1.Set(&wantUDP)
	//r1.SetSeq(1)
	// err := SignV4(&r1,privkey)
	//fmt.Println(err)
	//pyRecord ,err := r1.Encode(nil)
	//fmt.Println(err,hex.EncodeToString(pyRecord))
	var r enr.Record
	if _, err := r.Decode(pyRecord); err != nil {
		t.Fatalf("can't decode: %v", err)
	}
	n, err := New(ValidSchemes, &r)
	if err != nil {
		t.Fatalf("can't verify record: %v", err)
	}

	if n.Seq() != wantSeq {
		t.Errorf("wrong seq: got %d, want %d", n.Seq(), wantSeq)
	}
	if n.ID() != wantID {
		t.Errorf("wrong id: got %x, want %x", n.ID(), wantID)
	}
	want := map[enr.Entry]interface{}{new(enr.IP): &wantIP, new(enr.UDP): &wantUDP}
	for k, v := range want {
		desc := fmt.Sprintf("loading key %q", k.ENRKey())
		if assert.NoError(t, n.Load(k), desc) {
			assert.Equal(t, k, v, desc)
		}
	}
}
