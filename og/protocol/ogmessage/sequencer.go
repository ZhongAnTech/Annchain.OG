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
package ogmessage

import (
	"fmt"
	"github.com/annchain/OG/common"
	"strings"

	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp

////msgp:tuple Sequencer
//type Sequencer struct {
//	// TODO: need more states in sequencer to differentiate multiple chains
//	TxBase
//	Issuer         *common.Address
//	Signature    hexutil.Bytes
//	PublicKey hexutil.Bytes
//	StateRoot      common.Hash
//	Proposing      bool `msg:"-"` // is the sequencer is proposal ,did't commit yet ,use this flag to avoid bls sig verification failed
//}

//msgp:tuple SequencerJson
type SequencerJson struct {
	TxBaseJson
	Issuer         *common.Address `json:"issuer"`
	BlsJointSig    hexutil.Bytes   `json:"bls_joint_sig"`
	BlsJointPubKey hexutil.Bytes   `json:"bls_joint_pub_key"`
	Proposing      bool            `msg:"-",json:"-"`
}

//func (s *Sequencer) ToSmallCaseJson() ([]byte, error) {
//	if s == nil {
//		return nil, nil
//	}
//	j := SequencerJson{
//		TxBaseJson:     *s.TxBase.ToSmallCase(),
//		Issuer:         s.Issuer,
//		Signature:    s.Signature,
//		PublicKey: s.PublicKey,
//		Proposing:      s.Proposing,
//	}
//
//	return json.Marshal(&j)
//}

//msgp:tuple BlsSigSet
type BlsSigSet struct {
	PublicKey    []byte
	BlsSignature []byte
}

//msgp:tuple Sequencers
type Sequencers []*Sequencer

func (t *Sequencer) GetHead() *SequencerHeader {
	return NewSequencerHead(t.GetTxHash(), t.Height)
}

func (t *Sequencer) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("pHash:[%s], Issuer : %s , Height: %d, Weight: %d, nonce : %d , blspub: %s, signatute : %s, pubkey:  %s root: %s",
		strings.Join(phashes, " ,"),
		t.Issuer.Hex(),
		t.Height,
		t.Weight,
		t.AccountNonce,
		t.BlsJointPubKey,
		hexutil.Encode(t.PublicKey),
		hexutil.Encode(t.Signature),
		t.StateRoot.Hex(),
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
		StateRoot:      s.StateRoot,
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

func (t *Sequencer) SetSender(addr common.Address) {
	t.Issuer = &addr
}

func (r *Sequencers) Len() int {
	if r == nil {
		return 0
	}
	return len(*r)
}
