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
package p2p_message

import (
	"crypto/sha256"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/msg"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
)

//go:generate msgp

// HeightRound is the current progress of the consensus.
// Height is the block height, round is the sub-progress if no consensus can be easily reached
//msgp:tuple HeightRound
type HeightRound struct {
	Height uint64
	Round  int
}

func (h *HeightRound) String() string {
	return fmt.Sprintf("[%d-%d]", h.Height, h.Round)
}

// IsAfter judges whether self is a higher HeightRound
func (h *HeightRound) IsAfter(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round > o.Round)
}

// IsAfterOrEqual judges whether self is a higher or equal HeightRound
func (h *HeightRound) IsAfterOrEqual(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round >= o.Round)
}

// IsAfterOrEqual judges whether self is a lower HeightRound
func (h *HeightRound) IsBefore(o HeightRound) bool {
	return h.Height < o.Height ||
		(h.Height == o.Height && h.Round < o.Round)
}

type Proposal interface {
	Equal(Proposal) bool
	GetId() *common.Hash
	msg.Message
	String() string
	Copy() Proposal
}

//StringProposal is for test
type StringProposal string

func (s StringProposal) Equal(o Proposal) bool {
	v, ok := o.(*StringProposal)
	if !ok || v == nil {
		return false
	}
	return s == *v
}

func (s StringProposal) Copy() Proposal {
	var r StringProposal
	r = s
	return &r
}

func (s StringProposal) GetId() *common.Hash {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	hash := common.Hash{}
	hash.MustSetBytes(sum, common.PaddingNone)
	return &hash
}

func (s StringProposal) String() string {
	return string(s)
}

//msgp:tuple SequencerProposal
type SequencerProposal struct {
	tx_types.Sequencer
}

//msgp:tuple BasicMessage
type BasicMessage struct {
	SourceId    uint16
	HeightRound HeightRound
	TermId      uint32
}

//msgp:tuple MessageProposal
type MessageProposal struct {
	BasicMessage
	Value      Proposal //TODO
	ValidRound int
	//PublicKey  []byte
	Signature hexutil.Bytes
}

func (m MessageProposal) Copy() *MessageProposal {
	var r MessageProposal
	r = m
	r.Value = m.Value.Copy()
	return &r
}

//msgp:tuple MessageCommonVote
type MessagePreVote struct {
	BasicMessage
	Idv       *common.Hash // ID of the proposal, usually be the hash of the proposal
	Signature hexutil.Bytes
	PublicKey hexutil.Bytes
}

//msgp:tuple MessagePreCommit
type MessagePreCommit struct {
	BasicMessage
	Idv          *common.Hash // ID of the proposal, usually be the hash of the proposal
	BlsSignature hexutil.Bytes
	Signature    hexutil.Bytes
	PublicKey    hexutil.Bytes
}

func (m BasicMessage) String() string {
	return fmt.Sprintf("SourceId:%d ,hr:%d", m.SourceId, m.HeightRound)
}

func (m MessageProposal) String() string {
	return fmt.Sprintf("bm %s, value %s, vaildRound %d", m.BasicMessage, m.Value, m.ValidRound)
}

func (m MessagePreVote) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BasicMessage, m.Idv)
}

func (m MessagePreCommit) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BasicMessage, m.Idv)
}

func (m *MessagePreVote) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	if m.Idv != nil {
		w.Write(m.Idv.Bytes)
	}
	w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId)
	return w.Bytes()
}

func (m *MessagePreCommit) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	w.Write(m.BlsSignature)
	return w.Bytes()
}

func (m *MessagePreCommit) BlsSignatureTargets() []byte {
	w := types.NewBinaryWriter()
	if m.Idv != nil {
		w.Write(m.Idv.Bytes)
	}
	w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId)
	return w.Bytes()
}

func (m *MessageProposal) SignatureTargets() []byte {
	w := types.NewBinaryWriter()
	if idv := m.Value.GetId(); idv != nil {
		w.Write(idv.Bytes)
	}
	w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId, uint64(m.ValidRound))
	return w.Bytes()
}

func (s *SequencerProposal) String() string {
	return fmt.Sprintf("seqProposal") + s.Sequencer.String()
}

func (s SequencerProposal) Equal(o Proposal) bool {
	v, ok := o.(*SequencerProposal)
	if !ok || v == nil {
		return false
	}
	return s.GetTxHash() == v.GetTxHash()
}

func (s SequencerProposal) GetId() *common.Hash {
	//should copy ?
	var hash common.Hash
	hash.MustSetBytes(s.GetTxHash().ToBytes(), common.PaddingNone)
	return &hash
}

func (s SequencerProposal) Copy() Proposal {
	var r SequencerProposal
	r = s
	return &r
}
