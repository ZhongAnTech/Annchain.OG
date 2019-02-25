package tendermint

import (
	"time"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/ffchan"
	"fmt"
)

type ByzantineFeatures struct {
	SilenceProposal  bool
	SilencePreVote   bool
	SilencePreCommit bool
	BadProposal      bool
	BadPreVote       bool
	BadPreCommit     bool
}

// ByzantinePartner implements a Tendermint client according to "The latest gossip on BFT consensus"
type ByzantinePartner struct {
	PartnerBase
	Height    int
	Round     int
	N         int // total number of participants
	F         int // max number of Byzantines
	step      StepType
	Decisions map[int]interface{}
	Peers     []Partner
	quit      chan bool
	waiter    *Waiter

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
	ByzantineFeatures ByzantineFeatures
}

func (p *ByzantinePartner) SetPeers(peers []Partner) {
	p.Peers = peers
}

func (p *ByzantinePartner) resetStatus() {
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

func NewByzantinePartner(nbParticipants int, id int, byzantineFeatures ByzantineFeatures) *ByzantinePartner {
	p := &ByzantinePartner{
		N:                 nbParticipants,
		F:                 (nbParticipants - 1) / 3,
		Decisions:         make(map[int]interface{}),
		quit:              make(chan bool),
		ByzantineFeatures: byzantineFeatures,
		PartnerBase: PartnerBase{
			Id:                     id,
			IncomingMessageChannel: make(chan Message, 100),
			OutgoingMessageChannel: make(chan Message, 100),
			WaiterTimeoutChannel:   make(chan *WaiterRequest, 100),
		},
	}
	p.waiter = NewWaiter(p.GetWaiterTimeoutChannel())
	p.resetStatus()
	go p.waiter.StartEventLoop()
	return p
}

func (p *ByzantinePartner) StartRound(round int) {
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

func (p *ByzantinePartner) EventLoop() {
	go p.send()
	go p.receive()
}
func (p *ByzantinePartner) GetWaiterTimeoutChannel() chan *WaiterRequest {
	return p.WaiterTimeoutChannel
}

func (p *ByzantinePartner) GetIncomingMessageChannel() chan Message {
	return p.IncomingMessageChannel
}

func (p *ByzantinePartner) GetOutgoingMessageChannel() chan Message {
	return p.OutgoingMessageChannel
}

func (p *ByzantinePartner) GetId() int {
	return p.Id
}

func (p *ByzantinePartner) send() {
	timer := time.NewTimer(time.Second * 15)
	for {
		timer.Reset(time.Second * 15)
		select {
		case <-p.quit:
			break
		case <-timer.C:
			logrus.WithField("IM", p.Id).Warn("Blocked reading outgoing")
		case msg := <-p.OutgoingMessageChannel:
			msg, toSend := p.doBadThings(msg)
			if !toSend {
				continue
			}
			for _, peer := range p.Peers {
				logrus.WithFields(logrus.Fields{
					"IM":  p.Id,
					"to":  peer.GetId(),
					"msg": msg.String(),
				}).Info("Outgoing message")
				ffchan.NewTimeoutSenderShort(peer.GetIncomingMessageChannel(), msg, "")
			}
		}
	}
}
func (p *ByzantinePartner) doBadThings(msg Message) (updatedMessage Message, toSend bool) {
	updatedMessage = msg
	toSend = true
	switch msg.Type {
	case MessageTypeProposal:
		if p.ByzantineFeatures.SilenceProposal {
			toSend = false
		} else if p.ByzantineFeatures.BadProposal {
			v := updatedMessage.Payload.(MessageProposal)
			v.Round ++
			updatedMessage.Payload = v
		}

	case MessageTypePreVote:
		if p.ByzantineFeatures.SilencePreVote {
			toSend = false
		} else if p.ByzantineFeatures.BadPreVote {
			v := updatedMessage.Payload.(MessageCommonVote)
			v.Round ++
			updatedMessage.Payload = v
		}
	case MessageTypePreCommit:
		if p.ByzantineFeatures.SilencePreCommit {
			toSend = false
		} else if p.ByzantineFeatures.BadPreCommit {
			v := updatedMessage.Payload.(MessageCommonVote)
			v.Round ++
			updatedMessage.Payload = v
		}

	}
	return
}

func (p *ByzantinePartner) receive() {
	timer := time.NewTimer(time.Second * 15)
	for {
		timer.Reset(time.Second * 15)
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
func (p *ByzantinePartner) Proposer(height int, round int) int {
	return (height + round) % p.N
}

// GetValue generates the value requiring consensus
func (p *ByzantinePartner) GetValue() Proposal {
	time.Sleep(time.Millisecond * 100)
	v := fmt.Sprintf("[[[%d %d]]]", p.Height, p.Round)
	return StringProposal(v)
}

func (p *ByzantinePartner) Broadcast(messageType MessageType, height int, round int, content Proposal, validRound int) {
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

func (p *ByzantinePartner) OnTimeoutPropose(context WaiterContext) {
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
func (p *ByzantinePartner) OnTimeoutPreVote(context WaiterContext) {
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
func (p *ByzantinePartner) OnTimeoutPreCommit(context WaiterContext) {
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
func (p *ByzantinePartner) WaitStepTimeout(stepType StepType, timeout time.Duration, round int, timeoutCallback func(WaiterContext)) {
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

func (p *ByzantinePartner) handleMessage(message Message) {
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
func (p *ByzantinePartner) handleProposal(proposal *MessageProposal) {
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
func (p *ByzantinePartner) handlePreVote(vote *MessageCommonVote) {
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
func (p *ByzantinePartner) handlePreCommit(commit *MessageCommonVote) {
	// TODO：After +2/3 precommits for <nil>. --> goto Propose(H,R+1)
	// This rule is not in the paper but here: hhttps://github.com/tendermint/tendermint/wiki/Byzantine-Consensus-Algorithm#precommit-step-heighthroundr

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
func (p *ByzantinePartner) valid(proposal Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *ByzantinePartner) count(messageType MessageType, height int, validRound int, valueIdMatchType ValueIdMatchType, valueId string) int {
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
func (p *ByzantinePartner) checkRound(message *BasicMessage) {
	if message.Height == p.Height && message.Round > p.Round {
		// update round
		p.higherRoundCounter ++
	}
	if p.higherRoundCounter >= p.F+1 {
		p.StartRound(p.Round)
	}
}

func (p *ByzantinePartner) changeState(stepType StepType) {
	p.step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		Round:    p.Round,
		Height:   p.Height,
		StepType: stepType,
	})
}
