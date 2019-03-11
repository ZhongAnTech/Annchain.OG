package tendermint

import (
	"fmt"
	"time"
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

func (m StepType) IsAfter(o StepType) bool {
	return m > o
}

type MessageType int

func (m MessageType) String() string {
	switch m {
	case MessageTypeProposal:
		return "Proposal"
	case MessageTypePreVote:
		return "PreVote"
	case MessageTypePreCommit:
		return "PreCommit"
	default:
		return "Unknown"
	}
}

const (
	MessageTypeProposal MessageType = iota
	MessageTypePreVote
	MessageTypePreCommit
)

const (
	TimeoutPropose   = time.Duration(10) * time.Second
	TimeoutPreVote   = time.Duration(10) * time.Second
	TimeoutPreCommit = time.Duration(10) * time.Second
)

type ValueIdMatchType int

const (
	MatchTypeAny ValueIdMatchType = iota
	MatchTypeByValue
	MatchTypeNil
)

type Message struct {
	Type    MessageType
	Payload interface{}
}

func (m *Message) String() string {
	return fmt.Sprintf("%s %+v", m.Type.String(), m.Payload)
}

type Proposal interface {
	Equal(Proposal) bool
	GetId() string
}

type StringProposal string

func (s StringProposal) Equal(o Proposal) bool {
	v, ok := o.(StringProposal)
	if !ok {
		return false
	}
	return s == v
}

func (s StringProposal) GetId() string {
	return string(s)
}

type BasicMessage struct {
	SourceId    int
	HeightRound HeightRound
}
type MessageProposal struct {
	BasicMessage
	Value      Proposal
	ValidRound int
}
type MessageCommonVote struct {
	BasicMessage
	Idv string // ID of the proposal, usually be the hash of the proposal
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
