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
package sig_test

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"testing"
)

func TestSig(t *testing.T) {
	data, _ := hexutil.Decode("0x606060")
	pk, _ := crypto.PublicKeyFromString("0x010490e3c98826e0a530e22d34075e31cf478fead6297654dc5ce7e082fe1a29a3450ca669572bd257d2874c52158180b1cad654dbc091e92c40a983ee07361a17a7")

	tx := types.Tx{
		TxBase: types.TxBase{
			Type:         types.TxBaseTypeNormal,
			AccountNonce: 2,
			Hash:         types.HexToHash("0x89ed081994209f2d59fc51e4548e82c104e72f09f48eb1570a9baf00abb1341c"),
			PublicKey:    pk.Bytes,
		},
		From:  types.HexToAddress("0x49fdaab0af739e16c9e1c9bf1715a6503edf4cab"),
		Value: math.NewBigInt(0),
		To:    types.Address{},
		Data:  data,
	}
	fmt.Println(hexutil.Encode(tx.SignatureTargets()))
	fmt.Printf("%x ", tx.Data)

}
