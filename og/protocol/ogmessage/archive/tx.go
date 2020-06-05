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
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"math/rand"
	"strings"
	"time"

	"github.com/annchain/OG/common/math"
)

//go:generate msgp

//msgp:tuple Txs
type Txs []*Tx

//msgp:tuple Tx
type Tx struct {
	TxBase
	From    *common.Address
	To      common.Address
	Value   *math.BigInt
	TokenId int32
	Data    []byte
	confirm time.Time
}

func (t *Tx) GetConfirm() time.Duration {
	return time.Since(t.confirm)
}

func (t *Tx) Setconfirm() {
	t.confirm = time.Now()
}

func (t *Tx) String() string {
	if t.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-%d-Tx", t.TxBase.String(), t.AccountNonce)
	} else {
		return fmt.Sprintf("%s-[%.10s]-%d-Tx", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
	}

}

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)
	from := common.HexToAddress("0x99")
	return &Tx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  types.Hashes{types.HexToHash("0xCCDD"), types.HexToHash("0xEEFF")},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From:  &from,
		To:    common.HexToAddress("0x88"),
		Value: v,
	}
}

func RandomTx() *Tx {
	from := types.RandomAddress()
	return &Tx{TxBase: TxBase{
		Hash:         types.RandomHash(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  types.Hashes{types.RandomHash(), types.RandomHash()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From:  &from,
		To:    types.RandomAddress(),
		Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *Tx) Sender() common.Address {
	return *t.From
}

func (t *Tx) GetSender() *common.Address {
	return t.From
}

func (t *Tx) SetSender(addr common.Address) {
	t.From = &addr
}

func (t *Tx) RawTx() *RawTx {
	if t == nil {
		return nil
	}
	rawTx := &RawTx{
		TxBase:  t.TxBase,
		To:      t.To,
		Value:   t.Value,
		Data:    t.Data,
		TokenId: t.TokenId,
	}
	return rawTx
}

func (t Txs) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t Txs) ToRawTxs() RawTxs {
	if len(t) == 0 {
		return nil
	}
	var rawTxs []*RawTx
	for _, v := range t {
		rasTx := v.RawTx()
		rawTxs = append(rawTxs, rasTx)
	}
	return rawTxs
}

func (r *Txs) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}

func (c *Tx) RawTxi() RawTxi {
	return c.RawTx()
}
