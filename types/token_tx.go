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
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/common/msg"
	"math/rand"
	"strings"
	"time"
)

////go:generate msgp  never generate automaticly
const (
	ActionTxActionIPO uint8 = iota
	ActionTxActionWithdraw
	ActionTxActionSPO
	ActionRequestDomainName
)

type ActionData interface {
	msg.Message
	String() string
}

//msgp:tuple PublicOffering
type PublicOffering struct {
	TokenId int32 //for Secondary Public Offering
	Value   *math.BigInt
	//To      Address       //when publish a token ,to equals from
	EnableSPO bool //if enableSPO is false  , no Secondary Public Offering.
	TokenName string
}

//msgp:tuple RequestDomain
type RequestDomain struct {
	DomainName string
}

//msgp:tuple ActionTx
type ActionTx struct {
	TxBase
	Action     uint8
	From       *Address
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
	if t.GetSender()==nil {
		return fmt.Sprintf("%s-[nil]-%d-ATX", t.TxBase.String(), t.AccountNonce)
	}
	return fmt.Sprintf("%s-[%.10s]-%d-ATX", t.TxBase.String(), t.Sender(), t.AccountNonce)
}

func SampleActionTx() *ActionTx {
	//v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)
 from := HexToAddress("0x99")
	return &ActionTx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  Hashes{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From:&from ,
		//To:    HexToAddress("0x88"),
		//Value: v,
	}
}

func RandomActionTx() *ActionTx {
	from:=randomAddress()
	return &ActionTx{TxBase: TxBase{
		Hash:         randomHash(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  Hashes{randomHash(), randomHash()},
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From: &from,
		//To:    randomAddress(),
		//Value: math.NewBigInt(rand.Int63()),
	}
}

func (t *ActionTx) GetPublicOffering() *PublicOffering {
	if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionWithdraw {
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
	case ActionTxActionWithdraw:
	case ActionRequestDomainName:
	default:
		return false
	}
	return true
}

func (t *ActionTx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	if !Signer.CanRecoverPubFromSig() {
		panicIfError(binary.Write(&buf, binary.BigEndian, t.From.Bytes))
	}
	//panicIfError(binary.Write(&buf, binary.BigEndian, t.To.Bytes))
	if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionWithdraw {
		of := t.GetPublicOffering()
		panicIfError(binary.Write(&buf, binary.BigEndian, of.Value.GetSigBytes()))
		if t.Action == ActionTxActionIPO {
			panicIfError(binary.Write(&buf, binary.BigEndian, of.TokenId))
		}
		panicIfError(binary.Write(&buf, binary.BigEndian, of.EnableSPO))
	} else if t.Action == ActionRequestDomainName {
		r := t.GetDomainName()
		panicIfError(binary.Write(&buf, binary.BigEndian, r.DomainName))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Action))
	return buf.Bytes()
}

func (t *ActionTx) Sender() Address {
	return *t.From
}

func (tc *ActionTx) GetSender() *Address {
	return tc.From
}


func (t *ActionTx) GetOfferValue() *math.BigInt {
	return t.GetPublicOffering().Value
}

func (t *ActionTx) Parents() Hashes {
	return t.ParentsHash
}

func (t *ActionTx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *ActionTx:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
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

func (c*ActionTx)SetSender(addr Address) {
	c.From = &addr
}