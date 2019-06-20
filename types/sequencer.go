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
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/msg"
	"go.dedis.ch/kyber/v3/util/random"
	"math/rand"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

//msgp:tuple Sequencer
type Sequencer struct {
	// TODO: need more states in sequencer to differentiate multiple chains
	TxBase
	Issuer         Address
	BlsJointSig    msg.Bytes
	BlsJointPubKey msg.Bytes
	Proposing      bool `msg:"-"` // is the sequencer is proposal ,did't commit yet ,use this flag to avoid bls sig verification failed
}

type SequencerJson struct {
	TxBaseJson
	Issuer         Address   `json:"issuer"`
	BlsJointSig    msg.Bytes `json:"bls_joint_sig"`
	BlsJointPubKey msg.Bytes `json:"bls_joint_pub_key"`
	Proposing      bool      `msg:"-",json:"-"`
}

func (s *Sequencer) ToSmallCaseJson() ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	j := SequencerJson{
		TxBaseJson:     *s.TxBase.ToSmallCase(),
		Issuer:         s.Issuer,
		BlsJointSig:    s.BlsJointSig,
		BlsJointPubKey: s.BlsJointPubKey,
		Proposing:      s.Proposing,
	}

	return json.Marshal(&j)
}

func (t *Sequencer) String() string {
	return fmt.Sprintf("%s-[%.10s]-%d-Seq", t.TxBase.String(), t.Sender().String(), t.AccountNonce)
}

//msgp:tuple BlsSigSet
type BlsSigSet struct {
	PublicKey    []byte
	BlsSignature []byte
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
	r := random.New()
	seq := &Sequencer{
		TxBase: TxBase{
			Hash:         randomHash(),
			Height:       id,
			ParentsHash:  Hashes{randomHash(), randomHash()},
			Type:         TxBaseTypeSequencer,
			AccountNonce: uint64(rand.Int63n(50000)),
			Weight:       uint64(rand.Int31n(2000)),
		},
		Issuer: randomAddress(),
	}
	seq.BlsJointSig = make([]byte, 64)
	seq.BlsJointPubKey = make([]byte, 128)
	random.Bytes(seq.BlsJointPubKey, r)
	random.Bytes(seq.BlsJointSig, r)
	return seq
}

func (t *Sequencer) SignatureTargets() []byte {
	var buf bytes.Buffer

	panicIfError(binary.Write(&buf, binary.BigEndian, t.BlsJointPubKey))
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
	return fmt.Sprintf("pHash:[%s], Issuer : %s , Height: %d, Weight: %d, nonce : %d , blspub: %s, signatute : %s, pubkey:  %s",
		strings.Join(phashes, " ,"),
		t.Issuer.Hex(),
		t.Height,
		t.Weight,
		t.AccountNonce,
		t.BlsJointPubKey,
		hexutil.Encode(t.PublicKey),
		hexutil.Encode(t.Signature),
	)
}

func (s *Sequencer) RawSequencer() *RawSequencer {
	if s == nil {
		return nil
	}
	return &RawSequencer{
		TxBase:         s.TxBase,
		BlsJointSig:    s.BlsJointSig,
		BlsJointPubKey: s.BlsJointPubKey,
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
