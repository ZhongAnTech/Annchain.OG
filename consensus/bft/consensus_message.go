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
package bft

import (
	"crypto/sha256"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/msg"
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

//msgp:tuple BftMessageType
type BftMessageType uint16

func (m BftMessageType) String() string {
	switch m {
	case BftMessageTypeProposal:
		return "BFTProposal"
	case BftMessageTypePreVote:
		return "BFTPreVote"
	case BftMessageTypePreCommit:
		return "BFTPreCommit"
	default:
		return "BFTUnknown"
	}
}

const (
	BftMessageTypeProposal BftMessageType = iota + 100
	BftMessageTypePreVote
	BftMessageTypePreCommit
)

//msgp:tuple BftBasicInfo
type BftBasicInfo struct {
	SourceId    uint16
	HeightRound HeightRound
}

//msgp:tuple MessageProposal
type MessageProposal struct {
	BftBasicInfo
	Value      Proposal //TODO
	ValidRound int
}

func (m MessageProposal) Copy() *MessageProposal {
	var r MessageProposal
	r = m
	r.Value = m.Value.Copy()
	return &r
}

//msgp:tuple MessagePreVote
type MessagePreVote struct {
	BftBasicInfo
	Idv *common.Hash // ID of the proposal, usually be the hash of the proposal
}

//msgp:tuple MessagePreCommit
type MessagePreCommit struct {
	BftBasicInfo
	Idv          *common.Hash // ID of the proposal, usually be the hash of the proposal
	BlsSignature hexutil.Bytes
}

func (m BftBasicInfo) String() string {
	return fmt.Sprintf("SourceId:%d, hr:%d", m.SourceId, m.HeightRound)
}

func (m MessageProposal) String() string {
	return fmt.Sprintf("bm %s, value %s", m.BftBasicInfo, m.Value)
}

func (m MessagePreVote) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BftBasicInfo, m.Idv)
}

func (m MessagePreCommit) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BftBasicInfo, m.Idv)
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
	//w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId, uint64(m.ValidRound))
	w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId)
	return w.Bytes()
}

type Proposal interface {
	msg.MsgpMember
	Equal(Proposal) bool
	GetId() *common.Hash
	String() string
	Copy() Proposal
}

//StringProposal is for test
//msgp:tuple StringProposal
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

type ConsensusDecision Proposal

type Signable interface {
	msg.MsgpMember
	SignatureTargets() []byte
	String() string
}

//msgp:tuple BftMessage
type BftMessage struct {
	Type      BftMessageType
	PublicKey hexutil.Bytes
	Payload   Signable
}

func (m *BftMessage) String() string {
	return fmt.Sprintf("%s %+v", m.Type.String(), m.Payload)
}
