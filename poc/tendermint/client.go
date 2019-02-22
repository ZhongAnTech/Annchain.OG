package tendermint

import (
	"time"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/ffchan"
	"fmt"
)

// Partner implements a Tendermint client according to "The latest gossip on BFT consensus"
type Partner struct {
	Height    int
	Round     int
	Id        int
	N         int // total number of participants
	F         int // max number of Byzantines
	step      StepType
	Decisions map[int]interface{}
	Peers     []*Partner

	IncomingMessageChannel chan Message
	OutgoingMessageChannel chan Message
	quit                   chan bool
	waiter                 *Waiter

	// clear below every height
	MessageProposal *MessageProposal
	LockedValue     Proposal
	LockedRound     int
	validValue      Proposal
	validRound      int
	preVotes        []*MessageCommonVote
	preCommits      []*MessageCommonVote

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
	p.preVotes = make([]*MessageCommonVote, p.N)
	p.preCommits = make([]*MessageCommonVote, p.N)
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
		IncomingMessageChannel: make(chan Message, 100),
		OutgoingMessageChannel: make(chan Message, 100),
		quit:                   make(chan bool),
		waiter:                 NewWaiter(),
	}
	p.resetStatus()
	go p.waiter.StartEventLoop()
	return p
}

func (p *Partner) StartRound(round int) {
	p.Round = round
	p.changeState(StepTypePropose)
	p.higherRoundCounter = 0
	if p.Id == p.Proposer(p.Height, p.Round) {
		logrus.WithField("IM", p.Id).WithField("height", p.Height).WithField("round", p.Round).Info("I'm the proposer")
		var proposal Proposal
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
	go p.send()
	go p.receive()
}
func (p *Partner) send() {
	timer := time.NewTimer(time.Second * 5)
	for {
		timer.Reset(time.Second * 5)
		select {
		case <-p.quit:
			break
		case <-timer.C:
			logrus.WithField("id", p.Id).Warn("Blocked reading outgoing")
		case msg := <-p.OutgoingMessageChannel:
			for _, peer := range p.Peers {
				logrus.WithFields(logrus.Fields{
					"IM":  p.Id,
					"to":  peer.Id,
					"msg": msg.String(),
				}).Debug("Outgoing message")
				ffchan.NewTimeoutSenderShort(peer.IncomingMessageChannel, msg, "")
			}
		}
	}
}

func (p *Partner) receive() {
	timer := time.NewTimer(time.Second * 5)
	for {
		timer.Reset(time.Second * 5)
		select {
		case <-p.quit:
			break
		case <-timer.C:
			logrus.WithField("IM", p.Id).Warn("Blocked reading incoming")
		case msg := <-p.IncomingMessageChannel:
			logrus.WithFields(logrus.Fields{
				"IM":  p.Id,
				"msg": msg.String(),
			}).Debug("Incoming message")
			p.handleMessage(msg)
		}
	}
}

// Proposer returns current round proposer. Now simply round robin
func (p *Partner) Proposer(height int, round int) int {
	return (height + round) % p.N
}

// GetValue generates the value requiring consensus
func (p *Partner) GetValue() Proposal {
	time.Sleep(time.Millisecond * 1)
	v := fmt.Sprintf("[[[%d %d]]]",p.Height, p.Round)
	return StringProposal(v)
}

func (p *Partner) Broadcast(messageType MessageType, height int, round int, content Proposal, validRound int) {
	m := Message{
		Type: messageType,
	}
	basicMessage := BasicMessage{
		Height:   height,
		Round:    round,
		SourceId: p.Id,
	}
	idv := ""
	if content != nil {
		idv = content.GetId()
	}
	switch messageType {
	case MessageTypeProposal:
		m.Payload = MessageProposal{
			BasicMessage: basicMessage,
			Value:        content,
			ValidRound:   validRound,
		}
	case MessageTypePreVote:
		m.Payload = MessageCommonVote{
			BasicMessage: basicMessage,
			Idv:          idv,
		}
	case MessageTypePreCommit:
		m.Payload = MessageCommonVote{
			BasicMessage: basicMessage,
			Idv:          idv,
		}
	}
	ffchan.NewTimeoutSenderShort(p.OutgoingMessageChannel, m, "")
}

func (p *Partner) OnTimeoutPropose(context WaiterContext) {
	logrus.WithFields(logrus.Fields{
		"step":   StepTypePropose.String(),
		"IM":     p.Id,
		"height": p.Height,
		"round":  p.Round,
	}).Warn("wait step timeout")

	v := context.(*TendermintContext)
	if v.Height == p.Height && v.Round == p.Round && p.step == StepTypePropose {
		p.Broadcast(MessageTypePreVote, p.Height, p.Round, nil, 0)
		p.changeState(StepTypePreVote)
	}
}
func (p *Partner) OnTimeoutPreVote(context WaiterContext) {
	logrus.WithFields(logrus.Fields{
		"step":   StepTypePreVote.String(),
		"IM":     p.Id,
		"height": p.Height,
		"round":  p.Round,
	}).Warn("wait step timeout")
	v := context.(*TendermintContext)

	if v.Height == p.Height && v.Round == p.Round && p.step == StepTypePreVote {
		p.Broadcast(MessageTypePreCommit, p.Height, p.Round, nil, 0)
		p.changeState(StepTypePreCommit)
	}
}
func (p *Partner) OnTimeoutPreCommit(context WaiterContext) {
	logrus.WithFields(logrus.Fields{
		"step":   StepTypePreCommit.String(),
		"IM":     p.Id,
		"height": p.Height,
		"round":  p.Round,
	}).Warn("wait step timeout")
	v := context.(*TendermintContext)
	if v.Height == p.Height && v.Round == p.Round {
		p.StartRound(p.Round + 1)
	}
}

// WaitStepTimeout waits for a centain time if stepType stays too long
func (p *Partner) WaitStepTimeout(stepType StepType, timeout time.Duration, round int, timeoutCallback func(WaiterContext)) {
	p.waiter.UpdateRequest(&WaiterRequest{
		WaitTime:        timeout,
		TimeoutCallback: timeoutCallback,
		Context: &TendermintContext{
			Height:   p.Height,
			Round:    round,
			StepType: stepType,
		},
	})
}

func (p *Partner) handleMessage(message Message) {
	switch message.Type {
	case MessageTypeProposal:
		msg := message.Payload.(MessageProposal)
		p.checkRound(&msg.BasicMessage)
		p.handleProposal(&msg)
	case MessageTypePreVote:
		msg := message.Payload.(MessageCommonVote)
		p.preVotes[msg.SourceId] = &msg
		p.checkRound(&msg.BasicMessage)
		p.handlePreVote(&msg)
	case MessageTypePreCommit:
		msg := message.Payload.(MessageCommonVote)
		p.preCommits[msg.SourceId] = &msg
		p.checkRound(&msg.BasicMessage)
		p.handlePreCommit(&msg)
	}
}
func (p *Partner) handleProposal(proposal *MessageProposal) {
	p.MessageProposal = proposal
	// rule line 22
	if p.step == StepTypePropose {
		if p.valid(proposal.Value) && (p.LockedRound == -1 || p.LockedValue.Equal(proposal.Value)) {
			p.Broadcast(MessageTypePreVote, p.Height, p.Round, proposal.Value, 0)
		} else {
			p.Broadcast(MessageTypePreVote, p.Height, p.Round, nil, 0)
		}
		p.changeState(StepTypePreVote)
	}

	// rule line 28
	count := p.count(MessageTypePreVote, p.Height, proposal.ValidRound, MatchTypeByValue, proposal.Value.GetId())
	if count >= 2*p.F+1 {
		if p.step == StepTypePropose && (proposal.ValidRound >= 0 && proposal.ValidRound < p.Round) {
			if p.valid(proposal.Value) && (p.LockedRound <= proposal.ValidRound || p.LockedValue.Equal(proposal.Value)) {
				p.Broadcast(MessageTypePreVote, p.Height, p.Round, proposal.Value, 0)
			} else {
				p.Broadcast(MessageTypePreVote, p.Height, p.Round, nil, 0)
			}
			p.changeState(StepTypePreVote)
		}
	}
}
func (p *Partner) handlePreVote(vote *MessageCommonVote) {
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
				p.Broadcast(MessageTypePreCommit, p.Height, p.Round, p.MessageProposal.Value, 0)
				p.changeState(StepTypePreCommit)
			}
			p.validValue = p.MessageProposal.Value
			p.validRound = p.Round
		}
	}
	// rule line 44
	count = p.count(MessageTypePreVote, p.Height, p.Round, MatchTypeNil, "")
	if count >= 2*p.F+1 && p.step == StepTypePreVote {
		p.Broadcast(MessageTypePreCommit, p.Height, p.Round, nil, 0)
		p.changeState(StepTypePreCommit)
	}

}
func (p *Partner) handlePreCommit(commit *MessageCommonVote) {
	// rule line 47
	count := p.count(MessageTypePreCommit, p.Height, p.Round, MatchTypeAny, "")
	if count >= 2*p.F+1 && p.stepTypeEqualPreCommitFirstTime {
		p.stepTypeEqualPreCommitFirstTime = false
		p.WaitStepTimeout(StepTypePreCommit, TimeoutPreCommit, p.Round, p.OnTimeoutPreCommit)
	}
	// rule line 49
	if p.MessageProposal != nil {
		count = p.count(MessageTypePreCommit, p.Height, p.Round, MatchTypeByValue, p.MessageProposal.Value.GetId())
		if count >= 2*p.F+1 {
			if _, ok := p.Decisions[p.Height]; !ok {
				// output decision
				p.Decisions[p.Height] = p.MessageProposal.Value
				logrus.WithFields(logrus.Fields{
					"IM":     p.Id,
					"height": p.Height,
					"round":  p.Round,
					"value":  p.MessageProposal.Value,
				}).Info("Decision")
				p.Height ++
				p.resetStatus()
				p.StartRound(0)
			}
		}
	}

}

// check proposal validation
func (p *Partner) valid(proposal Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *Partner) count(messageType MessageType, height int, validRound int, valueIdMatchType ValueIdMatchType, valueId string) int {
	counter := 0
	var target []*MessageCommonVote
	switch messageType {
	case MessageTypePreVote:
		target = p.preVotes
	case MessageTypePreCommit:
		target = p.preCommits
	default:
		target = nil
	}
	for _, m := range target {
		if m == nil {
			continue
		}
		if m.Height != height || m.Round != validRound {
			continue
		}
		switch valueIdMatchType {
		case MatchTypeByValue:
			if m.Idv == valueId {
				counter ++
			}
		case MatchTypeNil:
			if m.Idv == "" {
				counter ++
			}
		case MatchTypeAny:
			counter ++
		}
	}
	return counter
}
func (p *Partner) checkRound(message *BasicMessage) {
	if message.Height == p.Height && message.Round > p.Round {
		// update round
		p.higherRoundCounter ++
	}
	if p.higherRoundCounter >= p.F+1 {
		p.StartRound(p.Round)
	}
}

func (p *Partner) changeState(stepType StepType) {
	p.step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		Round:    p.Round,
		Height:   p.Height,
		StepType: stepType,
	})
}
