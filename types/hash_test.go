package types

import "testing"

func TestHash(t *testing.T) {

	var emHash Hash
	var nHash Hash
	nHash = HexToHash("0xc770f1dccb00c0b845d36d3baee2590defee2d6894f853eb63a60270612271a3")
	if !emHash.Empty() {
		t.Fatalf("fail")
	}
	if nHash.Empty() {
		t.Fatalf("fail")
	}
}
