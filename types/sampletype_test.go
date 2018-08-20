package types

import (
	"testing"
	"fmt"
	"encoding/hex"
)

func TestSerializer(t *testing.T) {
	v := Foo{
		Bar:     "1234567890",                                           // 10 bytes
		Baz:     12.34213432423,                                         // 8 bytes
		Address: []byte{0x13, 0x13, 0x13, 0x13, 0x15, 0x13, 0x13, 0x15}, // 8 bytes
		Parents: []Hash{
			make([]byte, 0, 20),
			make([]byte, 0, 20),
			make([]byte, 0, 20),
			[]byte{0x0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // 20 bytes
		},
		KV: map[string]float64{
			"2test1": 43.3111,
			"41test1": 43.3111,
			"3test1": 43.3111,
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


func TestHasher(t *testing.T){
	v := Foo{
		Bar:     "1234567890",                                           // 10 bytes
		Baz:     12.34213432423,                                         // 8 bytes
		Address: []byte{0x13, 0x13, 0x13, 0x13, 0x15, 0x13, 0x13, 0x15}, // 8 bytes
		Parents: []Hash{
			make([]byte, 0, 20),
			make([]byte, 0, 20),
			make([]byte, 0, 20),
			[]byte{0x0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, // 20 bytes
		},
		KV: map[string]float64{
			"2test1": 43.3111,
			"41test1": 43.3111,
			"3test1": 43.3111,
			"41test2": 43.3111,
		},
	}

	v.CalcHash()
}