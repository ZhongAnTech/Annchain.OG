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
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/model"
	"github.com/annchain/OG/types"
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

//msgp:tuple BasicMessage
type BasicMessage struct {
	SourceId    uint16
	HeightRound HeightRound
}

//msgp:tuple MessageProposal
type MessageProposal struct {
	BasicMessage
	Value      model.Proposal //TODO
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
	BasicMessage
	Idv *common.Hash // ID of the proposal, usually be the hash of the proposal
}

//msgp:tuple MessagePreCommit
type MessagePreCommit struct {
	BasicMessage
	Idv          *common.Hash // ID of the proposal, usually be the hash of the proposal
	BlsSignature hexutil.Bytes
}

func (m BasicMessage) String() string {
	return fmt.Sprintf("SourceId:%d ,hr:%d", m.SourceId, m.HeightRound)
}

func (m MessageProposal) String() string {
	return fmt.Sprintf("bm %s, value %s", m.BasicMessage, m.Value)
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
	//w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId, uint64(m.ValidRound))
	w.Write(m.HeightRound.Height, uint64(m.HeightRound.Round), m.SourceId)
	return w.Bytes()
}
