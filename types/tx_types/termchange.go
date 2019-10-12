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
	"github.com/annchain/OG/og/protocol_message"
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

func (tc *TermChange) GetBase() *types.TxBase {
	return &tc.TxBase
}

func (tc *TermChange) Sender() common.Address {
	return *tc.Issuer
}

func (tc *TermChange) GetSender() *common.Address {
	return tc.Issuer
}

func (tc *TermChange) Compare(tx types.Txi) bool {
	switch tx := tx.(type) {
	case *TermChange:
		if tc.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (tc *TermChange) IsSameTermInfo(ctc *TermChange) bool {
	// compare term id.
	if tc.TermID != ctc.TermID {
		return false
	}
	// compare bls public key.
	if !common.IsSameBytes(tc.PkBls, ctc.PkBls) {
		return false
	}
	// compare sigset
	if len(tc.SigSet) != len(ctc.SigSet) {
		return false
	}
	oTcMap := map[string][]byte{}
	cTcMap := map[string][]byte{}
	for i := range tc.SigSet {
		oss := tc.SigSet[i]
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

func (tc *TermChange) Dump() string {
	var phashes []string
	for _, p := range tc.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	var sigs []string
	for _, v := range tc.SigSet {
		sigs = append(sigs, fmt.Sprintf("[pubkey: %x, sig: %x]", v.PublicKey, v.Signature))
	}
	return fmt.Sprintf("hash: %s, pHash: [%s], issuer: %s, nonce: %d , signatute: %s, pubkey %s ,"+
		"PkBls: %x, sigs: [%s]", tc.Hash.Hex(),
		strings.Join(phashes, ", "), tc.Issuer, tc.AccountNonce, hexutil.Encode(tc.Signature), hexutil.Encode(tc.PublicKey),
		tc.PkBls, strings.Join(sigs, ", "))
}

func (tc *TermChange) SignatureTargets() []byte {
	// add parents infornmation.
	w := types.NewBinaryWriter()

	w.Write(tc.AccountNonce)
	if !types.CanRecoverPubFromSig {
		w.Write(tc.Issuer.Bytes)
	}
	w.Write(tc.PkBls)
	for _, signatrue := range tc.SigSet {
		w.Write(signatrue.PublicKey, signatrue.Signature)
	}
	return w.Bytes()
}

func (tc *TermChange) String() string {
	if tc.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-id-%d-termChange", tc.TxBase.String(), tc.TermID)
	}
	return fmt.Sprintf("%s-[%.10s]-id-%d-termChange", tc.TxBase.String(), tc.Issuer.String(), tc.TermID)
}

func (c TermChanges) String() string {
	var strs []string
	for _, v := range c {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (c *TermChange) RawTermChange() *protocol_message.RawTermChange {
	if c == nil {
		return nil
	}
	rc := &protocol_message.RawTermChange{
		TxBase: c.TxBase,
		TermId: c.TermID,
		PkBls:  c.PkBls,
		SigSet: c.SigSet,
	}
	return rc
}

func (cs TermChanges) RawTermChanges() protocol_message.RawTermChanges {
	if len(cs) == 0 {
		return nil
	}
	var rawTcs protocol_message.RawTermChanges
	for _, v := range cs {
		rawTc := v.RawTermChange()
		rawTcs = append(rawTcs, rawTc)
	}
	return rawTcs
}

func (c *TermChange) RawTxi() types.RawTxi {
	return c.RawTermChange()
}

func (t *TermChange) SetSender(addr common.Address) {
	t.Issuer = &addr
}
func (r *TermChanges) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}