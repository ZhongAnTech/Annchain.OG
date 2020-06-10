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
package archive

import (
	"fmt"
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"

	"encoding/hex"

	"testing"
)

func TestSequencer(t *testing.T) {
	addr := types2.RandomAddress()
	from := common.HexToAddress("0x1")
	seq1 := Sequencer{TxBase: TxBase{ParentsHash: types2.Hashes{types2.HexToHash("0x0")}}, Issuer: &addr}
	seq2 := Tx{types.TxBase: TxBase{ParentsHash: types2.Hashes{types2.HexToHash("0x0")}},
		To:    common.HexToAddress("0x1"),
		From:  &from,
		Value: math.NewBigInt(0),
	}

	seq3 := Sequencer{TxBase: TxBase{ParentsHash: types2.Hashes{seq1.CalcMinedHash(), seq2.CalcMinedHash()}}}

	fmt.Println(seq1.CalcMinedHash().String())
	fmt.Println(seq2.CalcMinedHash().String())
	fmt.Println(seq3.CalcMinedHash().String())

}

func TestSequencerSize(t *testing.T) {
	addr := types2.RandomAddress()
	seq := Sequencer{Issuer: &addr}
	n := 100

	// make 1000 hashes
	fmt.Println("Length", seq.Msgsize(), seq.Msgsize()/n)
	bts, err := seq.MarshalMsg(nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(hex.Dump(bts))
}

func TestSequencerRawSize(t *testing.T) {
	addr := types2.RandomAddress()
	seq := Sequencer{Issuer: &addr}
	n := 100

	// make 1000 hashes
	for i := 0; i < n; i++ {
		//seq.Hashes = append(seq.Hashes, common.HexToHash("0xAABB000000000000000000000000CCDDCCDD000000000000000000000000EEFF").KeyBytes)
	}

	fmt.Println("Length", seq.Msgsize(), seq.Msgsize()/n)
	bts, err := seq.MarshalMsg(nil)
	fmt.Println("ActualLength", len(bts), len(bts)/n)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(hex.Dump(bts))
}

func TestSequencerSecpSign(t *testing.T) {
	t.Parallel()

	seq := Sequencer{}
	addr := common.HexToAddress("7349f7a6f622378d5fb0e2c16b9d4a3e5237c187")
	seq.Issuer = &addr
	seq.Height = 221

	// signer := ogcrypto.SignerSecp256k1{}

}

func TestSequencer_Number(t *testing.T) {
	seq := RandomSequencer()
	data, _ := seq.MarshalMsg(nil)
	var newSeq Sequencer
	o, err := newSeq.UnmarshalMsg(data)
	fmt.Println(o, err)
}
