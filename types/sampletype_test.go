package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/math"
	"github.com/tinylib/msgp/msgp"
	"testing"
)

func TestMap(t *testing.T) {
	v := map[Hash]int{}
	h1 := HexToHash("0xCC")
	v[h1] = 1
	h2 := HexToHash("0xCC")
	result, ok := v[h2]
	fmt.Println(result, ok)
}

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
		Seq: Sequencer{Id: 99,
			TxBase: TxBase{
				Height:       12,
				ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF"),},
				Type:         1,
				AccountNonce: 234,

			},
			ContractHashOrder: []Hash{
				HexToHash("0x00667788"),
				HexToHash("0xAA667788"),
				HexToHash("0xBB667788"), // 20 bytes
			},
		},
		TxInner: Tx{TxBase: TxBase{

			Height:       12,
			ParentsHash:  []Hash{HexToHash("0xCCDD"), HexToHash("0xEEFF"),},
			Type:         1,
			AccountNonce: 234,

		},
			From:  HexToAddress("0x99"),
			To:    HexToAddress("0x88"),
			Value: math.NewBigInt(54235432),
		},
	}
	fmt.Println(v.Msgsize())
	bts, err := v.MarshalMsg(nil)
	if err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	bufr := new(bytes.Buffer)
	bufr.Write(bts)

	msgp.CopyToJSON(buf, bufr)
	fmt.Println(jsonPrettyPrint(buf.String()))
	fmt.Println(hex.Dump(bts))
}

func jsonPrettyPrint(in string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(in), "", "\t")
	if err != nil {
		return in
	}
	return out.String()
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
