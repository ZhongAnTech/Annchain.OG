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
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"strings"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple TermChange
type TermChange struct {
	types.TxBase
	TermID uint64
	PkBls  hexutil.Bytes
	SigSet []*SigSet
	Issuer *common.Address
}

//msgp:tuple SigSet
type SigSet struct {
	PublicKey hexutil.Bytes
	Signature hexutil.Bytes
}

//msgp:tuple TermChanges
type TermChanges []*TermChange

func (t *TermChange) GetBase() *types.TxBase {
	return &t.TxBase
}

func (t *TermChange) Sender() common.Address {
	return *t.Issuer
}

func (t *TermChange) GetSender() *common.Address {
	return t.Issuer
}

func (t *TermChange) GetGuarantee() *math.BigInt {
	return nil
}

func (t *TermChange) Compare(tx types.Txi) bool {
	switch tx := tx.(type) {
	case *TermChange:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *TermChange) IsSameTermInfo(ctc *TermChange) bool {
	// compare term id.
	if t.TermID != ctc.TermID {
		return false
	}
	// compare bls public key.
	if !common.IsSameBytes(t.PkBls, ctc.PkBls) {
		return false
	}
	// compare sigset
	if len(t.SigSet) != len(ctc.SigSet) {
		return false
	}
	oTcMap := map[string][]byte{}
	cTcMap := map[string][]byte{}
	for i := range t.SigSet {
		oss := t.SigSet[i]
		oTcMap[common.Bytes2Hex(oss.PublicKey)] = oss.Signature
		css := ctc.SigSet[i]
		cTcMap[common.Bytes2Hex(css.PublicKey)] = css.Signature
	}
	for pk, os := range oTcMap {
		cs := cTcMap[pk]
		if cs == nil {
			return false
		}
		if !common.IsSameBytes(os, cs) {
			return false
		}
	}

	return true
}

func (t *TermChange) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	var sigs []string
	for _, v := range t.SigSet {
		sigs = append(sigs, fmt.Sprintf("[pubkey: %x, sig: %x]", v.PublicKey, v.Signature))
	}
	return fmt.Sprintf("hash: %s, pHash: [%s], issuer: %s, nonce: %d , signatute: %s, pubkey %s ,"+
		"PkBls: %x, sigs: [%s]", t.Hash.Hex(),
		strings.Join(phashes, ", "), t.Issuer, t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey),
		t.PkBls, strings.Join(sigs, ", "))
}

func (t *TermChange) SignatureTargets() []byte {
	// add parents infornmation.
	w := types.NewBinaryWriter()

	w.Write(t.AccountNonce)
	if !types.CanRecoverPubFromSig {
		w.Write(t.Issuer.Bytes)
	}
	w.Write(t.PkBls)
	for _, signatrue := range t.SigSet {
		w.Write(signatrue.PublicKey, signatrue.Signature)
	}
	return w.Bytes()
}

func (t *TermChange) String() string {
	if t.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-id-%d-termChange", t.TxBase.String(), t.TermID)
	}
	return fmt.Sprintf("%s-[%.10s]-id-%d-termChange", t.TxBase.String(), t.Issuer.String(), t.TermID)
}

func (c TermChanges) String() string {
	var strs []string
	for _, v := range c {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t *TermChange) RawTermChange() *RawTermChange {
	if t == nil {
		return nil
	}
	rc := &RawTermChange{
		TxBase: t.TxBase,
		TermId: t.TermID,
		PkBls:  t.PkBls,
		SigSet: t.SigSet,
	}
	return rc
}

func (cs TermChanges) RawTermChanges() RawTermChanges {
	if len(cs) == 0 {
		return nil
	}
	var rawTcs RawTermChanges
	for _, v := range cs {
		rawTc := v.RawTermChange()
		rawTcs = append(rawTcs, rawTc)
	}
	return rawTcs
}

func (t *TermChange) RawTxi() types.RawTxi {
	return t.RawTermChange()
}

func (t *TermChange) SetSender(addr common.Address) {
	t.Issuer = &addr
}
