package tendermint

import (
	"time"
	"fmt"
)

type StepType int

const (
	StepTypePropose   StepType = iota
	StepTypePreVote
	StepTypePreCommit
)

type MessageType int

const (
	MessageTypeProposal  MessageType = iota
	MessageTypePreVote
	MessageTypePreCommit
)

const (
	TimeoutPropose   = time.Duration(5) * time.Second
	TimeoutPreVote   = time.Duration(5) * time.Second
	TimeoutPreCommit = time.Duration(5) * time.Second
)

type ValueIdMatchType int

const (
	MatchTypeAny     ValueIdMatchType = iota
	MatchTypeByValue
	MatchTypeNil
)

type Message struct {
	Type    MessageType
	Payload interface{}
}
type Proposal interface {
	Equal(Proposal) bool
	GetId() string
}
type MessageProposal struct {
	SourceId   int
	Height     int
	Round      int
	Value      Proposal
	ValidRound int
}
type MessagePreVote struct {
	SourceId int
	Height   int
	Round    int
	Idv      string // ID of the proposal, usually be the hash of the proposal
}
type MessagePreCommit struct {
	SourceId int
	Height   int
	Round    int
	Idv      string // ID of the proposal, usually be the hash of the proposal
}

// Partner implements a Tendermint client according to "The latest gossip on BFT consensus"
type Partner struct {
	Height    int
	Round     int
	Id        int
	N         int // total number of participants
	F         int // max number of Byzantines
	step      StepType
	Decisions map[int]interface{}

	IncomingMessageChannel chan Message
	OutgoingMessageChannel chan Message
	quit                   chan bool

	// clear below every height
	MessageProposal                       *MessageProposal
	LockedValue                           Proposal
	LockedRound                           int
	validValue                            Proposal
	validRound                            int
	preVotes                              []MessagePreVote
	preCommit                             []MessagePreCommit
	stepTypeEqualPreVoteFirstTime         bool // for line 34
	stepTypeEqualOrLargerPreVoteFirstTime bool // for line 36
	stepTypeEqualPreCommitFirstTime       bool // for line 47
	higherRoundCounter                    int  // for line 55
	// consider updating resetStatus() if you want to add things here
}

func (p *Partner) resetStatus() {
	p.LockedRound = -1
	p.LockedValue = nil
	p.validRound = -1
	p.validValue = nil
	p.MessageProposal = nil
	p.preVotes = make([]MessagePreVote, p.N)
	p.preCommit = make([]MessagePreCommit, p.N)
	p.stepTypeEqualPreCommitFirstTime = true
	p.stepTypeEqualPreVoteFirstTime = true
	p.stepTypeEqualOrLargerPreVoteFirstTime = true
}

func NewPartner(nbParticipants int, id int) *Partner {
	p := &Partner{
		Id:                     id,
		N:                      nbParticipants,
		F:                      (nbParticipants - 1) / 3,
		Decisions:              make(map[int]interface{}),
		IncomingMessageChannel: make(chan Message),
		OutgoingMessageChannel: make(chan Message),
		quit:                   make(chan bool),
	}
	p.resetStatus()
	return p
}

func (p *Partner) StartRound(round int) {
	p.Round = round
	p.step = StepTypePropose
	p.higherRoundCounter = 0
	if p.Id == p.Proposer(p.Height, p.Round) {
		var proposal interface{}
		if p.validValue != nil {
			proposal = p.validValue
		} else {
			proposal = p.GetValue()
		}
		// broadcast
		p.Broadcast(MessageTypeProposal, p.Height, p.Round, proposal, p.validRound)
	} else {
		p.WaitStepTimeout(StepTypePropose, TimeoutPropose, p.Round, p.OnTimeoutPropose)
	}
}

func (p *Partner) EventLoop() {
	for {
		select {
		case <-p.quit:
			break
		case msg := <-p.IncomingMessageChannel:
			p.handleMessage(msg)
		}
	}

}

// Proposer returns current round proposer. Now simply round robin
func (p *Partner) Proposer(height int, round int) int {

}

// GetValue generates the value requiring consensus
func (p *Partner) GetValue() interface{} {
	v := time.Now().String()
	return v
}

func (p *Partner) Broadcast(messageType MessageType, height int, round int, content interface{}, validRound int) {

}

func (p *Partner) OnTimeoutPropose(height int, round int) {

}
func (p *Partner) OnTimeoutPreVote(height int, round int) {

}
func (p *Partner) OnTimeoutPreCommit(height int, round int) {

}
func (p *Partner) WaitStepTimeout(stepType StepType, timeout time.Duration, round int, timeoutCallback func(height int, round int)) {

}
func (p *Partner) handleMessage(message Message) {
	switch message.Type {
	case MessageTypeProposal:
		msg := message.Payload.(MessageProposal)
		p.checkRound(msg)
		p.handleProposal(&msg)
	case MessageTypePreVote:
		msg := message.Payload.(MessagePreVote)
		p.checkRound(msg)
		p.handlePreVote(&msg)
	case MessageTypePreCommit:
		msg := message.Payload.(MessagePreCommit)
		p.checkRound(msg)
		p.handlePreCommit(&msg)
	}
}
func (p *Partner) handleProposal(proposal *MessageProposal) {
	p.MessageProposal = proposal
	// rule line 22
	if p.step == StepTypePropose {
		if p.valid(proposal.Value) && (p.LockedRound == -1 || p.LockedValue.Equal(proposal.Value)) {
			p.Broadcast(MessageTypePreVote, p.Height, p.Round, proposal.Value.GetId(), 0)
		} else {
			p.Broadcast(MessageTypePreVote, p.Height, p.Round, nil, 0)
		}
		p.step = StepTypePreVote
	}

	// rule line 28
	count := p.count(MessageTypePreVote, p.Height, proposal.ValidRound, MatchTypeByValue, proposal.Value.GetId())
	if count >= 2*p.F+1 {
		if p.step == StepTypePropose && (proposal.ValidRound >= 0 && proposal.ValidRound < p.Round) {
			if p.valid(proposal.Value) && (p.LockedRound <= proposal.ValidRound || p.LockedValue.Equal(proposal)) {
				p.Broadcast(MessageTypePreVote, p.Height, p.Round, proposal.Value.GetId(), 0)
			} else {
				p.Broadcast(MessageTypePreVote, p.Height, p.Round, nil, 0)
			}
			p.step = StepTypePreVote
		}
	}
}
func (p *Partner) handlePreVote(vote *MessagePreVote) {
	// rule line 34
	count := p.count(MessageTypePreVote, p.Height, p.Round, MatchTypeAny, "")
	if count >= 2*p.F+1 {
		if p.step == StepTypePreVote && p.stepTypeEqualPreVoteFirstTime {
			p.stepTypeEqualPreVoteFirstTime = false
			p.WaitStepTimeout(StepTypePreVote, TimeoutPreVote, p.Round, p.OnTimeoutPreVote)
		}
	}
	// rule line 36
	if p.MessageProposal != nil && count >= 2*p.F+1 {
		if p.valid(p.MessageProposal.Value) && p.step >= StepTypePreVote && p.stepTypeEqualOrLargerPreVoteFirstTime {
			p.stepTypeEqualOrLargerPreVoteFirstTime = false
			if p.step == StepTypePreVote {
				p.LockedValue = p.MessageProposal.Value
				p.LockedRound = p.Round
				p.Broadcast(MessageTypePreCommit, p.Height, p.Round, p.MessageProposal.Value.GetId(), 0)
				p.step = StepTypePreCommit
			}
			p.validValue = p.MessageProposal.Value
			p.validRound = p.Round
		}
	}
	// rule line 44
	count = p.count(MessageTypePreVote, p.Height, p.Round, MatchTypeNil, "")
	if count >= 2*p.F+1 && p.step == StepTypePreVote {
		p.Broadcast(MessageTypePreCommit, p.Height, p.Round, nil, 0)
		p.step = StepTypePreCommit
	}

}
func (p *Partner) handlePreCommit(commit *MessagePreCommit) {
	// rule line 47
	count := p.count(MessageTypePreCommit, p.Height, p.Round, MatchTypeAny, "")
	if count >= 2*p.F+1 && p.stepTypeEqualPreCommitFirstTime {
		p.stepTypeEqualPreCommitFirstTime = false
		p.WaitStepTimeout(StepTypePreCommit, TimeoutPreCommit, p.Round, p.OnTimeoutPreCommit)
	}
	// rule line 49
	count = p.count(MessageTypePreCommit, p.Height, p.MessageProposal.Round, MatchTypeByValue, p.MessageProposal.Value.GetId())
	if p.MessageProposal != nil && count >= 2*p.F+1 {
		if v, ok := p.Decisions[p.Height]; !ok {
			// output decision
			p.Decisions[p.Height] = p.MessageProposal.Value
			p.Height ++
			fmt.Printf("Decision: %d %s", p.Height, v)
			p.resetStatus()
			p.StartRound(0)
		}
	}
}

// check proposal validation
func (p *Partner) valid(proposal Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *Partner) count(messageType MessageType, height int, validRound int, valueIdMatchType ValueIdMatchType, valueId string) int {

}
func (p *Partner) checkRound(proposal MessageProposal) {
	if proposal.Height == p.Height && proposal.Round > p.Round {
		// update round
		StartRound(p.Round)
	}
}
