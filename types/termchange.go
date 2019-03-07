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
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp
//msgp:tuple TermChange

type TermChange struct {
	TxBase
	PkBls  []byte
	SigSet map[Address][]byte
	Issuer Address
}

func (tc *TermChange) GetBase() *TxBase {
	return &tc.TxBase
}

func (tc *TermChange) Sender() Address {
	return tc.Issuer
}

func (tc *TermChange) Compare(tx Txi) bool {
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

func (tc *TermChange) Dump() string {
	var phashes []string
	for _, p := range tc.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	var sigs []string
	for k, v := range tc.SigSet {
		sigs = append(sigs, fmt.Sprintf("[addr: %x, sig: %x]", k.ToBytes(), v))
	}
	return fmt.Sprintf("hash: %s, pHash: [%s], issuer: %s, nonce: %d , signatute: %s, pubkey %s ,"+
		"PkBls: %x, sigs: [%s]", tc.Hash.Hex(),
		strings.Join(phashes, ", "), tc.Issuer.Hex(), tc.AccountNonce, hexutil.Encode(tc.Signature), hexutil.Encode(tc.PublicKey),
		tc.PkBls, strings.Join(sigs, ", "))
}

func (tc *TermChange) SignatureTargets() []byte {
	// TODO
	// add parents infornmation.
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, tc.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, tc.Issuer.Bytes))

	return buf.Bytes()
}

func (tc *TermChange)String() string {
	return fmt.Sprintf("%s-[%s]-termChange",tc.TxBase.String(),tc.Issuer.String())
}
