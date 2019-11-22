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
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types"
	"math/rand"
	"strings"
	"time"

	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp

//msgp:tuple Txs
type Txs []*Tx

//msgp:tuple Tx
type Tx struct {
	types.TxBase
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
	return &Tx{TxBase: types.TxBase{
		Height:       12,
		ParentsHash:  common.Hashes{common.HexToHash("0xCCDD"), common.HexToHash("0xEEFF")},
		Type:         types.TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From:  &from,
		To:    common.HexToAddress("0x88"),
		Value: v,
	}
}

func RandomTx() *Tx {
	from := common.RandomAddress()
	return &Tx{TxBase: types.TxBase{
		Hash:         common.RandomHash(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  common.Hashes{common.RandomHash(), common.RandomHash()},
		Type:         types.TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From:  &from,
		To:    common.RandomAddress(),
		Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *Tx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	w := types.NewBinaryWriter()

	w.Write(t.AccountNonce)
	if !types.CanRecoverPubFromSig {
		w.Write(t.From.Bytes)
	}
	w.Write(t.To.Bytes, t.Value.GetSigBytes(), t.Data, t.TokenId)
	return w.Bytes()
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

func (t *Tx) GetValue() *math.BigInt {
	return t.Value
}

func (t *Tx) Parents() common.Hashes {
	return t.ParentsHash
}

func (t *Tx) Compare(tx types.Txi) bool {
	switch tx := tx.(type) {
	case *Tx:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *Tx) GetBase() *types.TxBase {
	return &t.TxBase
}

func (t *Tx) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash %s, pHash:[%s], from : %s , to :%s ,value : %s ,\n nonce : %d , signatute : %s, pubkey: %s ,"+
		"height: %d , mined Nonce: %v, type: %v, weight: %d, data: %x", t.Hash.Hex(),
		strings.Join(phashes, " ,"), t.From.Hex(), t.To.Hex(), t.Value,
		t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey), t.Height, t.MineNonce, t.Type, t.Weight, t.Data)
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

func (c *Tx) RawTxi() types.RawTxi {
	return c.RawTx()
}

type TransactionMsg struct {
	Type    int      `json:"type"`
	Hash    string   `json:"hash"`
	Parents []string `json:"parents"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Nonce   uint64   `json:"nonce"`
	Value   string   `json:"value"`
	Weight  uint64   `json:"weight"`
}

func (t *Tx) ToJsonMsg() TransactionMsg {
	txMsg := TransactionMsg{}

	txMsg.Type = int(t.GetType())
	txMsg.Hash = t.GetTxHash().Hex()
	txMsg.From = t.From.Hex()
	txMsg.To = t.To.Hex()
	txMsg.Nonce = t.GetNonce()
	txMsg.Value = t.GetValue().String()
	txMsg.Weight = t.GetWeight()

	txMsg.Parents = make([]string, 0)
	for _, p := range t.ParentsHash {
		txMsg.Parents = append(txMsg.Parents, p.Hex())
	}

	return txMsg
}
