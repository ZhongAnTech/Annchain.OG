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
	"math/rand"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple Sequencer
type Sequencer struct {
	// TODO: need more states in sequencer to differentiate multiple chains
	TxBase
	Issuer Address
}

func (t *Sequencer) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d-Seq", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

//msgp:tuple Sequencers
type Sequencers []*Sequencer

func SampleSequencer() *Sequencer {
	return &Sequencer{
		TxBase: TxBase{
			Height:       12,
			ParentsHash:  Hashes{HexToHash("0xCCDD"), HexToHash("0xEEFF")},
			Type:         TxBaseTypeSequencer,
			AccountNonce: 234,
		},
		Issuer: HexToAddress("0x33"),
	}
}

func RandomSequencer() *Sequencer {
	id := uint64(rand.Int63n(1000))

	return &Sequencer{TxBase: TxBase{
		Hash:         randomHash(),
		Height:       id,
		ParentsHash:  Hashes{randomHash(), randomHash()},
		Type:         TxBaseTypeSequencer,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		Issuer: randomAddress(),
	}
}

func (t *Sequencer) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.AccountNonce))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Issuer.Bytes))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, t.Weight))
	for _, parent := range t.Parents() {
		panicIfError(binary.Write(&buf, binary.BigEndian, parent.Bytes))
	}
	return buf.Bytes()
}

func (t *Sequencer) Sender() Address {
	return t.Issuer
}

func (t *Sequencer) Parents() Hashes {
	return t.ParentsHash
}

func (t *Sequencer) Number() uint64 {
	return t.GetHeight()
}

func (t *Sequencer) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Sequencer:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *Sequencer) GetBase() *TxBase {
	return &t.TxBase
}

func (t *Sequencer) GetHead() *SequencerHeader {
	return NewSequencerHead(t.GetTxHash(), t.Height)
}

func (t *Sequencer) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("pHash:[%s], Issuer : %s , Height :%d , nonce : %d , signatute : %s, pubkey %s",
		strings.Join(phashes, " ,"), t.Issuer.Hex(), t.Height,
		t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey))
}

func (s *Sequencer) RawSequencer() *RawSequencer {
	if s == nil {
		return nil
	}
	return &RawSequencer{
		TxBase: s.TxBase,
	}
}

func (s Sequencers) String() string {
	var strs []string
	for _, v := range s {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (s Sequencers) ToHeaders() SequencerHeaders {
	if len(s) == 0 {
		return nil
	}
	var headers SequencerHeaders
	for _, v := range s {
		head := NewSequencerHead(v.Hash, v.Height)
		headers = append(headers, head)
	}
	return headers
}

func (seqs Sequencers) ToRawSequencers() RawSequencers {
	if len(seqs) == 0 {
		return nil
	}
	var rawSeqs RawSequencers
	for _, v := range seqs {
		rasSeq := v.RawSequencer()
		rawSeqs = append(rawSeqs, rasSeq)
	}
	return rawSeqs
}

func (c *Sequencer) RawTxi() RawTxi {
	return c.RawSequencer()
}
