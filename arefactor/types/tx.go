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
	"github.com/annchain/commongo/marshaller"
	"math/rand"
	"strings"
	"time"

	"github.com/annchain/commongo/hexutil"
	"github.com/annchain/commongo/math"

	og_types "github.com/annchain/OG/og_interface"
)

//go:generate msgp

//msgp:tuple Txs
type Txs []*Tx

//msgp:tuple Tx
type Tx struct {
	TxBase
	From    og_types.Address
	To      og_types.Address
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
		return fmt.Sprintf("%s-[%.10s]-%d-Tx", t.TxBase.String(), t.Sender().AddressShortString(), t.AccountNonce)
	}

}

func SampleTx() *Tx {
	v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)
	from, _ := og_types.HexToAddress20("0x99")
	to, _ := og_types.HexToAddress20("0x88")

	hash1, _ := og_types.HexToHash32("0xCCDD")
	hash2, _ := og_types.HexToHash32("0xEEFF")

	return &Tx{
		TxBase: TxBase{
			Height:       12,
			ParentsHash:  []og_types.Hash{hash1, hash2},
			Type:         TxBaseTypeNormal,
			AccountNonce: 234,
		},
		From:  from,
		To:    to,
		Value: v,
	}
}

func RandomTx() *Tx {
	return &Tx{TxBase: TxBase{
		Hash:         og_types.RandomHash32(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  []og_types.Hash{og_types.RandomHash32(), og_types.RandomHash32()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From:  og_types.RandomAddress20(),
		To:    og_types.RandomAddress20(),
		Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *Tx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	w := NewBinaryWriter()

	w.Write(t.AccountNonce)
	if !CanRecoverPubFromSig {
		w.Write(t.From.Bytes)
	}
	w.Write(t.To.Bytes, t.Value.GetSigBytes(), t.Data, t.TokenId)
	return w.Bytes()
}

func (t *Tx) Sender() og_types.Address {
	return t.From
}

func (t *Tx) GetSender() og_types.Address {
	return t.From
}

func (t *Tx) SetSender(addr og_types.Address) {
	t.From = addr
}

func (t *Tx) GetValue() *math.BigInt {
	return t.Value
}

func (t *Tx) Parents() []og_types.Hash {
	return t.ParentsHash
}

func (t *Tx) Compare(tx Txi) bool {
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

func (t *Tx) GetBase() *TxBase {
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

/**
marshaller part
*/

func (t *Tx) MarshalMsg() ([]byte, error) {
	var err error

	b := make([]byte, marshaller.HeaderSize)

	// (byte) TokenID
	b = marshaller.AppendInt32(b, t.TokenId)

	// (BigInt) Value
	b = marshaller.AppendBigInt(b, t.Value.Value)

	// (Address) From, To
	fromB, err := t.From.MarshalMsg()
	if err != nil {
		return nil, err
	}
	b = append(b, fromB...)

	toB, err := t.To.MarshalMsg()
	if err != nil {
		return nil, err
	}
	b = append(b, toB...)

	// ([]byte) Data
	b = marshaller.AppendBytes(b, t.Data)

	// (TxBase) TxBase
	baseB, err := t.TxBase.MarshalMsg()
	if err != nil {
		return nil, err
	}
	b = append(b, baseB...)

	// fill in header bytes
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (t *Tx) UnmarshalMsg(b []byte) ([]byte, error) {
	var err error

	b, _, err = marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}

	t.TokenId, b, err = marshaller.ReadInt32(b)
	if err != nil {
		return nil, err
	}

	valueB, b, err := marshaller.ReadBigInt(b)
	if err != nil {
		return nil, err
	}
	t.Value = math.NewBigIntFromBigInt(valueB)

	t.From, b, err = og_types.UnmarshalAddress(b)
	if err != nil {
		return nil, err
	}

	t.To, b, err = og_types.UnmarshalAddress(b)
	if err != nil {
		return nil, err
	}

	t.Data, b, err = marshaller.ReadBytes(b)
	if err != nil {
		return nil, err
	}

	t.TxBase = TxBase{}
	b, err = t.TxBase.UnmarshalMsg(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (t *Tx) MsgSize() int {
	return marshaller.Int32Size + // TokenID
		marshaller.CalIMarshallerSize(len(t.Value.GetBytes())) + // Value
		marshaller.CalIMarshallerSize(t.From.MsgSize()) + // From
		marshaller.CalIMarshallerSize(t.To.MsgSize()) + // To
		marshaller.CalIMarshallerSize(len(t.Data)) + // Data
		marshaller.CalIMarshallerSize(t.TxBase.MsgSize()) // TxVase
}

func (t Txs) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

//func (t Txs) ToRawTxs() RawTxs {
//	if len(t) == 0 {
//		return nil
//	}
//	var rawTxs []*RawTx
//	for _, v := range t {
//		rasTx := v.RawTx()
//		rawTxs = append(rawTxs, rasTx)
//	}
//	return rawTxs
//}

//func (c *Tx) RawTxi() RawTxi {
//	return c.RawTx()
//}

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
