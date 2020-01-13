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
	"github.com/annchain/OG/common/byteutil"
	"math/rand"
	"strings"
	"time"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types/msg"
)

//go:generate msgp  never generate automaticly
const (
	ActionTxActionIPO uint8 = iota
	ActionTxActionDestroy
	ActionTxActionSPO
	ActionRequestDomainName
)

type ActionData interface {
	msg.MsgpMember
	String() string
}

//msgp:tuple PublicOffering
type PublicOffering struct {
	TokenId int32        `json:"token_id"` //for Secondary Public Issues
	Value   *math.BigInt `json:"value"`
	//To      Address       //when publish a token ,to equals from
	EnableSPO bool   `json:"enable_spo"` //if enableSPO is false  , no Secondary Public Issues.
	TokenName string `json:"token_name"`
}

func NewPublicOffering() *PublicOffering {
	return &PublicOffering{
		Value: math.NewBigInt(0),
	}
}

//msgp:tuple RequestDomain
type RequestDomain struct {
	DomainName string
}

//msgp:tuple ActionTx
type ActionTx struct {
	TxBase
	Action     uint8
	From       *common.Address
	ActionData ActionData
	confirm    time.Time
}

func (p PublicOffering) String() string {
	return fmt.Sprintf("tokenid %d,value %v, EnableSPO %v", p.TokenId, p.Value, p.EnableSPO)
}

func (r RequestDomain) String() string {
	return r.DomainName
}

func (t *ActionTx) GetConfirm() time.Duration {
	return time.Since(t.confirm)
}

func (t *ActionTx) Setconfirm() {
	t.confirm = time.Now()
}

func (t *ActionTx) String() string {
	if t.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-%d-ATX", t.TxBase.String(), t.AccountNonce)
	}
	return fmt.Sprintf("%s-[%.10s]-%d-ATX", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

func SampleActionTx() *ActionTx {
	//v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)
	from := common.HexToAddress("0x99")
	return &ActionTx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  common.Hashes{common.HexToHash("0xCCDD"), common.HexToHash("0xEEFF")},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From: &from,
		//To:    common.HexToAddress("0x88"),
		//Value: v,
	}
}

func RandomActionTx() *ActionTx {
	from := common.RandomAddress()
	return &ActionTx{TxBase: TxBase{
		Hash:         common.RandomHash(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  common.Hashes{common.RandomHash(), common.RandomHash()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From: &from,
		//To:     common.RandomAddress(),
		//Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *ActionTx) GetPublicOffering() *PublicOffering {
	if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionDestroy {
		v, ok := t.ActionData.(*PublicOffering)
		if ok {
			return v
		}
	}
	return nil
}

func (t *ActionTx) GetDomainName() *RequestDomain {
	if t.Action == ActionRequestDomainName {
		v, ok := t.ActionData.(*RequestDomain)
		if ok {
			return v
		}
	}
	return nil
}

func (t *ActionTx) CheckActionIsValid() bool {
	switch t.Action {
	case ActionTxActionIPO:
	case ActionTxActionSPO:
	case ActionTxActionDestroy:
	case ActionRequestDomainName:
	default:
		return false
	}
	return true
}

func (t *ActionTx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	w := byteutil.NewBinaryWriter()

	w.Write(t.AccountNonce, t.Action)
	if !CanRecoverPubFromSig {
		w.Write(t.From.Bytes)
	}
	//types.PanicIfError(binary.Write(&buf, binary.BigEndian, t.To.KeyBytes))
	if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionDestroy {
		of := t.GetPublicOffering()
		w.Write(of.Value.GetSigBytes(), of.EnableSPO)
		if t.Action == ActionTxActionIPO {
			w.Write([]byte(of.TokenName))
		} else {
			w.Write(of.TokenId)
		}

	} else if t.Action == ActionRequestDomainName {
		r := t.GetDomainName()
		w.Write(r.DomainName)
	}
	return w.Bytes()
}

func (t *ActionTx) Sender() common.Address {
	return *t.From
}

func (tc *ActionTx) GetSender() *common.Address {
	return tc.From
}

func (t *ActionTx) GetOfferValue() *math.BigInt {
	return t.GetPublicOffering().Value
}

func (t *ActionTx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *ActionTx:
		if t.GetHash().Cmp(tx.GetHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *ActionTx) GetBase() *TxBase {
	return &t.TxBase
}

func (t *ActionTx) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash %s, pHash:[%s], from : %s  \n nonce : %d , signatute : %s, pubkey: %s ,"+
		"height: %d , mined Nonce: %v, type: %v, weight: %d, action %d, actionData %s", t.Hash.Hex(),
		strings.Join(phashes, " ,"), t.From.Hex(),
		t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey),
		t.Height, t.MineNonce, t.Type, t.Weight, t.Action, t.ActionData)
}
func (t *ActionTx) RawActionTx() *RawActionTx {
	if t == nil {
		return nil
	}
	rawTx := &RawActionTx{
		TxBase:     t.TxBase,
		Action:     t.Action,
		ActionData: t.ActionData,
	}
	return rawTx
}

type ActionTxs []*ActionTx

func (t ActionTxs) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t ActionTxs) ToRawTxs() RawActionTxs {
	if len(t) == 0 {
		return nil
	}
	var rawTxs RawActionTxs
	for _, v := range t {
		rasTx := v.RawActionTx()
		rawTxs = append(rawTxs, rasTx)
	}
	return rawTxs
}

func (c *ActionTx) RawTxi() RawTxi {
	return c.RawActionTx()
}

func (c *ActionTx) SetSender(addr common.Address) {
	c.From = &addr
}

func (r *ActionTxs) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}
