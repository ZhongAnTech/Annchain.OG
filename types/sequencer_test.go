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
package types

import (
	"fmt"

	"github.com/annchain/OG/common/math"

	"encoding/hex"

	"testing"
)

func TestSequencer(t *testing.T) {
	seq1 := Sequencer{TxBase: TxBase{ParentsHash: Hashes{HexToHash("0x0")}}, Issuer: randomAddress()}
	seq2 := Tx{TxBase: TxBase{ParentsHash: Hashes{HexToHash("0x0")}},
		To:    HexToAddress("0x1"),
		From:  HexToAddress("0x1"),
		Value: math.NewBigInt(0),
	}

	seq3 := Sequencer{TxBase: TxBase{ParentsHash: Hashes{seq1.CalcMinedHash(), seq2.CalcMinedHash()}}}

	fmt.Println(seq1.CalcMinedHash().String())
	fmt.Println(seq2.CalcMinedHash().String())
	fmt.Println(seq3.CalcMinedHash().String())

}

func TestSequencerSize(t *testing.T) {
	seq := Sequencer{Issuer: randomAddress()}
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
	seq := Sequencer{Issuer: randomAddress()}
	n := 100

	// make 1000 hashes
	for i := 0; i < n; i++ {
		//seq.Hashes = append(seq.Hashes, HexToHash("0xAABB000000000000000000000000CCDDCCDD000000000000000000000000EEFF").Bytes)
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
	seq.Issuer = HexToAddress("7349f7a6f622378d5fb0e2c16b9d4a3e5237c187")
	seq.Height = 221

	// signer := crypto.SignerSecp256k1{}

}
