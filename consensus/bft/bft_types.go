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
	"encoding/json"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/model"
	"time"
)

const (
	TimeoutPropose   = time.Duration(8) * time.Second
	TimeoutPreVote   = time.Duration(8) * time.Second
	TimeoutPreCommit = time.Duration(8) * time.Second
	TimeoutDelta     = time.Duration(1) * time.Second
)

type ValueIdMatchType int

const (
	MatchTypeAny ValueIdMatchType = iota
	MatchTypeByValue
	MatchTypeNil
)

type StepType int

const (
	StepTypePropose StepType = iota
	StepTypePreVote
	StepTypePreCommit
)

func (m StepType) String() string {
	switch m {
	case StepTypePropose:
		return "Proposal"
	case StepTypePreVote:
		return "PreVote"
	case StepTypePreCommit:
		return "PreCommit"
	default:
		return "Unknown"
	}
}

func (m *StepType) MarshalJSON() ([]byte, error) {
	s := m.String()
	return json.Marshal(&s)
}

func (m StepType) IsAfter(o StepType) bool {
	return m > o
}

type ChangeStateEvent struct {
	NewStepType StepType
	HeightRound HeightRound
}

type TendermintContext struct {
	HeightRound HeightRound
	StepType    StepType
}

func (t *TendermintContext) Equal(w WaiterContext) bool {
	v, ok := w.(*TendermintContext)
	if !ok {
		return false
	}
	return t.HeightRound == v.HeightRound && t.StepType == v.StepType
}

func (t *TendermintContext) IsAfter(w WaiterContext) bool {
	v, ok := w.(*TendermintContext)
	if !ok {
		return false
	}
	return t.HeightRound.IsAfter(v.HeightRound) || (t.HeightRound == v.HeightRound && t.StepType.IsAfter(v.StepType))
}

type PeerInfo struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

// HeightRoundState is the structure for each Height/Round
// Always keep this state that is higher than current in Partner.States map in order not to miss future things
type HeightRoundState struct {
	MessageProposal                       *MessageProposal // the proposal received in this round
	LockedValue                           model.Proposal
	LockedRound                           int
	ValidValue                            model.Proposal
	ValidRound                            int
	Decision                              model.ConsensusDecision // final decision of mine in this round
	PreVotes                              []*MessagePreVote       // other peers' PreVotes
	PreCommits                            []*MessagePreCommit     // other peers' PreCommits
	Sources                               map[uint16]bool         // for line 55, who send future round so that I may advance?
	StepTypeEqualPreVoteTriggered         bool                    // for line 34, FIRST time trigger
	StepTypeEqualOrLargerPreVoteTriggered bool                    // for line 36, FIRST time trigger
	StepTypeEqualPreCommitTriggered       bool                    // for line 47, FIRST time trigger
	Step                                  StepType                // current step in this round
	StartAt                               time.Time
}

func NewHeightRoundState(total int) *HeightRoundState {
	return &HeightRoundState{
		LockedRound: -1,
		ValidRound:  -1,
		PreVotes:    make([]*MessagePreVote, total),
		PreCommits:  make([]*MessagePreCommit, total),
		Sources:     make(map[uint16]bool),
		StartAt:     time.Now(),
	}
}

type HeightRoundStateMap map[HeightRound]*HeightRoundState

func (h *HeightRoundStateMap) MarshalJSON() ([]byte, error) {
	if h == nil {
		return nil, nil
	}
	m := make(map[string]*HeightRoundState, len(*h))
	for k, v := range *h {
		m[k.String()] = v
	}
	return json.Marshal(&m)
}

// BftStatus records all states of BFT
// consider updating resetStatus() if you want to add things here
type BftStatus struct {
	CurrentHR HeightRound
	N         int // total number of participants
	F         int // max number of Byzantines
	Maj23     int
	Peers     []PeerInfo
	States    HeightRoundStateMap // for line 55, round number -> count
}
