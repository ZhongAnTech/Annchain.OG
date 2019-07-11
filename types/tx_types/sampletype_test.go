// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tx_types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/tinylib/msgp/msgp"
	"testing"
)

func TestMap(t *testing.T) {
	v := map[common.Hash]int{}
	h1 := common.HexToHash("0xCC")
	v[h1] = 1
	h2 := common.HexToHash("0xCC")
	result, ok := v[h2]
	fmt.Println(result, ok)
}

func TestSerializer(t *testing.T) {
	from := common.HexToAddress("0x99")
	v := Foo{
		Bar:     "1234567890",                      // 10 bytes
		Baz:     12.34213432423,                    // 8 bytes
		Address: common.HexToAddress("0x11223344"), // 4 bytes
		Parents: common.Hashes{
			common.HexToHash("0x00667788"),
			common.HexToHash("0xAA667788"),
			common.HexToHash("0xBB667788"), // 20 bytes
		},
		KV: map[string]float64{
			"2test1":  43.3111,
			"41test1": 43.3111,
			"3test1":  43.3111,
			"41test2": 43.3111,
		},
		Seq: Sequencer{
			TxBase: types.TxBase{
				Height:       12,
				ParentsHash:  common.Hashes{common.HexToHash("0xCCDD"), common.HexToHash("0xEEFF")},
				Type:         1,
				AccountNonce: 234,
			},
		},
		TxInner: Tx{TxBase: types.TxBase{

			Height:       12,
			ParentsHash:  common.Hashes{common.HexToHash("0xCCDD"), common.HexToHash("0xEEFF")},
			Type:         1,
			AccountNonce: 234,
		},
			From:  &from,
			To:    common.HexToAddress("0x88"),
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
		Bar:     "1234567890",                      // 10 bytes
		Baz:     12.34213432423,                    // 8 bytes
		Address: common.HexToAddress("0x11223344"), // 4 bytes
		Parents: common.Hashes{
			common.HexToHash("0x00667788"),
			common.HexToHash("0xAA667788"),
			common.HexToHash("0xBB667788"), // 20 bytes
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
