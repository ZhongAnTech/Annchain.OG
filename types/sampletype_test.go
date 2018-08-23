package types

import (
	"testing"
	"fmt"
	"encoding/hex"
)

func TestSerializer(t *testing.T) {
	v := Foo{
		Bar:     "1234567890",               // 10 bytes
		Baz:     12.34213432423,             // 8 bytes
		Address: HexToAddress("0x11223344"), // 4 bytes
		Parents: []Hash{
			HexToHash("0x00667788"),
			HexToHash("0xAA667788"),
			HexToHash("0xBB667788"), // 20 bytes
		},
		KV: map[string]float64{
			"2test1":  43.3111,
			"41test1": 43.3111,
			"3test1":  43.3111,
			"41test2": 43.3111,
		},
	}
	fmt.Println(v.Msgsize())
	bts, err := v.MarshalMsg(nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(hex.Dump(bts))
}

func TestHasher(t *testing.T) {
	v := Foo{
		Bar:     "1234567890",               // 10 bytes
		Baz:     12.34213432423,             // 8 bytes
		Address: HexToAddress("0x11223344"), // 4 bytes
		Parents: []Hash{
			HexToHash("0x00667788"),
			HexToHash("0xAA667788"),
			HexToHash("0xBB667788"), // 20 bytes
		},
		KV: map[string]float64{
			"2test1":  43.3111,
			"41test1": 43.3111,
			"3test1":  43.3111,
			"41test2": 43.3111,
		},
	}

	v.CalcHash()
}
