package tendermint

import (
	"fmt"
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Partner interface {
	EventLoop()
	StartNewEra(height int, round int)
	SetPeers(peers []Partner)
	GetIncomingMessageChannel() chan Message
	GetOutgoingMessageChannel() chan Message
	GetWaiterTimeoutChannel() chan *WaiterRequest
	GetId() int
}

type PartnerBase struct {
	Id                     int
	IncomingMessageChannel chan Message
	OutgoingMessageChannel chan Message
	WaiterTimeoutChannel   chan *WaiterRequest
}

type HeightRound struct {
	Height int
	Round  int
}

func (h *HeightRound) String() string {
	return fmt.Sprintf("[%d-%d]", h.Height, h.Round)
}
func (h *HeightRound) IsAfter(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round > o.Round)
}
func (h *HeightRound) IsAfterOrEqual(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round >= o.Round)
}
func (h *HeightRound) IsBefore(o HeightRound) bool {
	return h.Height < o.Height ||
		(h.Height == o.Height && h.Round < o.Round)
}

// HeightRoundState is the structure for each Height/Round
// Always keep this state that is higher than current in Partner.States map in order not to miss future things
type HeightRoundState struct {
	MessageProposal                       *MessageProposal
	LockedValue                           Proposal
	LockedRound                           int
	ValidValue                            Proposal
	ValidRound                            int
	Decision                              interface{}
	PreVotes                              []*MessageCommonVote
	PreCommits                            []*MessageCommonVote
	Sources                               map[int]bool // for line 55, who send future round so that I may advance?
	StepTypeEqualPreVoteTriggered         bool         // for line 34
	StepTypeEqualOrLargerPreVoteTriggered bool         // for line 36
	StepTypeEqualPreCommitTriggered       bool         // for line 47
	Step                                  StepType
}

func NewHeightRoundState(total int) *HeightRoundState {
	return &HeightRoundState{
		LockedRound: -1,
		ValidRound:  -1,
		PreVotes:    make([]*MessageCommonVote, total),
		PreCommits:  make([]*MessageCommonVote, total),
		Sources:     make(map[int]bool),
	}
}

// DefaultPartner implements a Tendermint client according to "The latest gossip on BFT consensus"
// Destroy and use a new one upon peers changing.
type DefaultPartner struct {
	PartnerBase
	CurrentHR HeightRound
	blockTime time.Duration
	N         int // total number of participants
	F         int // max number of Byzantines
	Peers     []Partner
	quit      chan bool
	waiter    *Waiter
	States    map[HeightRound]*HeightRoundState // for line 55, round number -> count
	// consider updating resetStatus() if you want to add things here
}

func (p *DefaultPartner) GetWaiterTimeoutChannel() chan *WaiterRequest {
	return p.WaiterTimeoutChannel
}

func (p *DefaultPartner) GetIncomingMessageChannel() chan Message {
	return p.IncomingMessageChannel
}

func (p *DefaultPartner) GetOutgoingMessageChannel() chan Message {
	return p.OutgoingMessageChannel
}

func (p *DefaultPartner) GetId() int {
	return p.Id
}

func (p *DefaultPartner) SetPeers(peers []Partner) {
	p.Peers = peers
}

func NewPartner(nbParticipants int, id int, blockTime time.Duration) *DefaultPartner {
	p := &DefaultPartner{
		N:         nbParticipants,
		F:         (nbParticipants - 1) / 3,
		blockTime: blockTime,
		PartnerBase: PartnerBase{
			Id:                     id,
			IncomingMessageChannel: make(chan Message, 10),
			OutgoingMessageChannel: make(chan Message, 10),
			WaiterTimeoutChannel:   make(chan *WaiterRequest, 10),
		},
		quit:   make(chan bool),
		States: make(map[HeightRound]*HeightRoundState),
	}
	p.waiter = NewWaiter(p.GetWaiterTimeoutChannel())
	go p.waiter.StartEventLoop()
	return p
}

func (p *DefaultPartner) StartNewEra(height int, round int) {
	hr := p.CurrentHR
	if  height - hr.Height > 1{
		logrus.WithField("height", height).Warn("height is much higher than current. Indicating packet loss or severe behind.")
	}
	hr.Height = height
	hr.Round = round

	logrus.WithFields(logrus.Fields{
		"IM":        p.Id,
		"currentHR": p.CurrentHR.String(),
		"newHR":     hr.String(),
	}).Debug("Starting new round")

	currState := p.initHeightRound(hr)
	// update partner height
	p.CurrentHR = hr

	p.WipeOldStates()
	p.changeState(StepTypePropose)

	if p.Id == p.Proposer(p.CurrentHR) {
		logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR.String()).Info("I'm the proposer")
		var proposal Proposal
		if currState.ValidValue != nil {
			proposal = currState.ValidValue
		} else {
			proposal = p.GetValue()
		}
		// broadcast
		p.Broadcast(MessageTypeProposal, p.CurrentHR, proposal, currState.ValidRound)
	} else {
		p.WaitStepTimeout(StepTypePropose, TimeoutPropose, p.CurrentHR, p.OnTimeoutPropose)
	}
}

func (p *DefaultPartner) EventLoop() {
	go p.send()
	go p.receive()
}

// send is just for outgoing messages. It should not change any state of local tendermint
func (p *DefaultPartner) send() {
	timer := time.NewTimer(time.Second * 7)
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			break
		case <-timer.C:
			logrus.WithField("IM", p.Id).Warn("Blocked reading outgoing")
			p.dumpAll("blocked reading outgoing")
		case msg := <-p.OutgoingMessageChannel:
			for _, peer := range p.Peers {
				logrus.WithFields(logrus.Fields{
					"IM":   p.Id,
					"hr":   p.CurrentHR.String(),
					"from": p.Id,
					"to":   peer.GetId(),
					"msg":  msg.String(),
				}).Debug("Out")
				ffchan.NewTimeoutSenderShort(peer.GetIncomingMessageChannel(), msg, "")
			}
		}
	}
}

// receive prevents concurrent issues by allowing only one channel to be read per loop
func (p *DefaultPartner) receive() {
	timer := time.NewTimer(time.Second * 7)
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			break
		case v := <-p.WaiterTimeoutChannel:
			context := v.Context.(*TendermintContext)
			logrus.WithFields(logrus.Fields{
				"step": context.StepType.String(),
				"IM":   p.Id,
				"hr":   context.HeightRound.String(),
			}).Warn("wait step timeout")
			p.dumpAll("wait step timeout")
			v.TimeoutCallback(v.Context)
		case <-timer.C:
			logrus.WithField("IM", p.Id).Warn("Blocked reading incoming")
			p.dumpAll("blocked reading incoming")
		case msg := <-p.IncomingMessageChannel:
			p.handleMessage(msg)
		}
	}
}

// Proposer returns current round proposer. Now simply round robin
func (p *DefaultPartner) Proposer(hr HeightRound) int {
	//return 0
	return (hr.Height + hr.Round) % p.N
}

// GetValue generates the value requiring consensus
func (p *DefaultPartner) GetValue() Proposal {
	logrus.Info("sleep")
	time.Sleep(p.blockTime)
	logrus.Info("sleep end")
	v := fmt.Sprintf("■■■%d %d■■■", p.CurrentHR.Height, p.CurrentHR.Round)
	return StringProposal(v)
}

func (p *DefaultPartner) Broadcast(messageType MessageType, hr HeightRound, content Proposal, validRound int) {
	m := Message{
		Type: messageType,
	}
	basicMessage := BasicMessage{
		HeightRound: hr,
		SourceId:    p.Id,
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

func (p *DefaultPartner) OnTimeoutPropose(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR && p.States[p.CurrentHR].Step == StepTypePropose {
		p.Broadcast(MessageTypePreVote, p.CurrentHR, nil, 0)
		p.changeState(StepTypePreVote)
	}
}
func (p *DefaultPartner) OnTimeoutPreVote(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR && p.States[p.CurrentHR].Step == StepTypePreVote {
		p.Broadcast(MessageTypePreCommit, p.CurrentHR, nil, 0)
		p.changeState(StepTypePreCommit)
	}
}
func (p *DefaultPartner) OnTimeoutPreCommit(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR {
		p.StartNewEra(v.HeightRound.Height, v.HeightRound.Round+1)
	}
}

// WaitStepTimeout waits for a centain time if stepType stays too long
func (p *DefaultPartner) WaitStepTimeout(stepType StepType, timeout time.Duration, hr HeightRound, timeoutCallback func(WaiterContext)) {
	p.waiter.UpdateRequest(&WaiterRequest{
		WaitTime:        timeout,
		TimeoutCallback: timeoutCallback,
		Context: &TendermintContext{
			HeightRound: hr,
			StepType:    stepType,
		},
	})
}

func (p *DefaultPartner) handleMessage(message Message) {
	// state := p.States[p.CurrentHR]
	switch message.Type {
	case MessageTypeProposal:
		msg := message.Payload.(MessageProposal)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			break
		}
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
			"value":  msg.Value,
		}).Debug("In")
		p.handleProposal(&msg)
	case MessageTypePreVote:
		msg := message.Payload.(MessageCommonVote)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			break
		}
		p.States[msg.HeightRound].PreVotes[msg.SourceId] = &msg
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreVote(&msg)
	case MessageTypePreCommit:
		msg := message.Payload.(MessageCommonVote)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			break
		}
		p.States[msg.HeightRound].PreCommits[msg.SourceId] = &msg
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreCommit(&msg)
	}
}
func (p *DefaultPartner) handleProposal(proposal *MessageProposal) {
	state, ok := p.States[proposal.HeightRound]
	if !ok {
		panic("must exists")
	}
	state.MessageProposal = proposal
	// rule line 22
	if state.Step == StepTypePropose {
		if p.valid(proposal.Value) && (state.LockedRound == -1 || state.LockedValue.Equal(proposal.Value)) {
			p.Broadcast(MessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
		} else {
			p.Broadcast(MessageTypePreVote, proposal.HeightRound, nil, 0)
		}
		p.changeState(StepTypePreVote)
	}

	// rule line 28
	count := p.count(MessageTypePreVote, proposal.HeightRound.Height, proposal.ValidRound, MatchTypeByValue, proposal.Value.GetId())
	if count >= 2*p.F+1 {
		if state.Step == StepTypePropose && (proposal.ValidRound >= 0 && proposal.ValidRound < p.CurrentHR.Round) {
			if p.valid(proposal.Value) && (state.LockedRound <= proposal.ValidRound || state.LockedValue.Equal(proposal.Value)) {
				p.Broadcast(MessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
			} else {
				p.Broadcast(MessageTypePreVote, proposal.HeightRound, nil, 0)
			}
			p.changeState(StepTypePreVote)
		}
	}
}
func (p *DefaultPartner) handlePreVote(vote *MessageCommonVote) {
	// rule line 34
	count := p.count(MessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeAny, "")
	state, ok := p.States[vote.HeightRound]
	if !ok {
		panic("should exists: " + vote.HeightRound.String())
	}
	if count >= 2*p.F+1 {
		if state.Step == StepTypePreVote && !state.StepTypeEqualPreVoteTriggered {
			logrus.WithField("IM", p.Id).WithField("hr", vote.HeightRound.String()).Debug("prevote counter is more than 2f+1 #1")
			state.StepTypeEqualPreVoteTriggered = true
			p.WaitStepTimeout(StepTypePreVote, TimeoutPreVote, vote.HeightRound, p.OnTimeoutPreVote)
		}
	}
	// rule line 36
	if state.MessageProposal != nil && count >= 2*p.F+1 {
		if p.valid(state.MessageProposal.Value) && state.Step >= StepTypePreVote && !state.StepTypeEqualOrLargerPreVoteTriggered {
			logrus.WithField("IM", p.Id).WithField("hr", vote.HeightRound.String()).Debug("prevote counter is more than 2f+1 #2")
			state.StepTypeEqualOrLargerPreVoteTriggered = true
			if state.Step == StepTypePreVote {
				state.LockedValue = state.MessageProposal.Value
				state.LockedRound = p.CurrentHR.Round
				p.Broadcast(MessageTypePreCommit, vote.HeightRound, state.MessageProposal.Value, 0)
				p.changeState(StepTypePreCommit)
			}
			state.ValidValue = state.MessageProposal.Value
			state.ValidRound = p.CurrentHR.Round
		}
	}
	// rule line 44
	count = p.count(MessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeNil, "")
	if count >= 2*p.F+1 && state.Step == StepTypePreVote {
		logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR.String()).Debug("prevote counter is more than 2f+1 #3")
		p.Broadcast(MessageTypePreCommit, vote.HeightRound, nil, 0)
		p.changeState(StepTypePreCommit)
	}

}
func (p *DefaultPartner) handlePreCommit(commit *MessageCommonVote) {
	// rule line 47
	count := p.count(MessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeAny, "")
	state := p.States[commit.HeightRound]
	if count >= 2*p.F+1 && !state.StepTypeEqualPreCommitTriggered {
		state.StepTypeEqualPreCommitTriggered = true
		p.WaitStepTimeout(StepTypePreCommit, TimeoutPreCommit, commit.HeightRound, p.OnTimeoutPreCommit)
	}
	// rule line 49
	if state.MessageProposal != nil {
		count = p.count(MessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeByValue, state.MessageProposal.Value.GetId())
		if count >= 2*p.F+1 {
			if state.Decision == nil {
				// output decision
				state.Decision = state.MessageProposal.Value
				logrus.WithFields(logrus.Fields{
					"IM":    p.Id,
					"hr":    p.CurrentHR.String(),
					"value": state.MessageProposal.Value,
				}).Info("Decision")
				p.StartNewEra(p.CurrentHR.Height+1, 0)
			}
		}
	}

}

// check proposal validation
func (p *DefaultPartner) valid(proposal Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *DefaultPartner) count(messageType MessageType, height int, validRound int, valueIdMatchType ValueIdMatchType, valueId string) int {
	counter := 0
	var target []*MessageCommonVote
	state, ok := p.States[HeightRound{
		Height: height,
		Round:  validRound,
	}]
	if !ok {
		return 0
	}
	switch messageType {
	case MessageTypePreVote:
		target = state.PreVotes
	case MessageTypePreCommit:
		target = state.PreCommits
	default:
		target = nil
	}
	for _, m := range target {
		if m == nil {
			continue
		}
		if m.HeightRound.Height > height || m.HeightRound.Round > validRound {
			p.dumpAll("impossible now")
			panic("wrong logic: " + fmt.Sprintf("%d %d %d %d", m.HeightRound.Height, height, m.HeightRound.Round, validRound))
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
	logrus.WithField("IM", p.Id).
		Debugf("Counting: [%d] %s H:%d VR:%d MT:%d", counter, messageType.String(), height, validRound, valueIdMatchType)
	return counter
}

// checkRound will init all data structure this message needs.
// It also check if the message is out of date, or advanced too much
func (p *DefaultPartner) checkRound(message *BasicMessage) (needHandle bool) {
	// rule line 55
	// slightly changed this so that if there is f+1 newer HeightRound(instead of just round), catch up to this HeightRound
	if message.HeightRound.IsAfter(p.CurrentHR) {
		state, ok := p.States[message.HeightRound]
		if !ok {
			// create one
			// TODO: verify if someone is generating garbage height
			state = p.initHeightRound(message.HeightRound)

		}
		state.Sources[message.SourceId] = true
		logrus.Infof("%d's %s state is %+v", p.Id, p.CurrentHR.String(), state.Sources)

		if len(state.Sources) >= p.F+1 {
			p.dumpAll("New era received")
			p.StartNewEra(message.HeightRound.Height, message.HeightRound.Round)
		}
	}
	return message.HeightRound.IsAfterOrEqual(p.CurrentHR)
}

func (p *DefaultPartner) changeState(stepType StepType) {
	p.States[p.CurrentHR].Step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		HeightRound: p.CurrentHR,
		StepType:    stepType,
	})
}

func (p *DefaultPartner) dumpVotes(votes []*MessageCommonVote) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for _, vote := range votes {
		if vote == nil {
			sb.WriteString(fmt.Sprintf("[nil Vote]"))
		} else {
			sb.WriteString(fmt.Sprintf("[%d hr:%s s:%s]", vote.SourceId, vote.HeightRound.String(), vote.Idv))
		}

		sb.WriteString(" ")
	}
	sb.WriteString("]")
	return sb.String()
}
func (p *DefaultPartner) dumpAll(reason string) {
	//return
	state := p.States[p.CurrentHR]
	logrus.WithField("IM", p.Id).WithField("reason", reason).Info("Dumping")
	logrus.WithField("IM", p.Id).WithField("votes", "prevotes").Info(p.dumpVotes(state.PreVotes))
	logrus.WithField("IM", p.Id).WithField("votes", "precommits").Info(p.dumpVotes(state.PreCommits))
	logrus.WithField("IM", p.Id).WithField("step", state.Step.String()).Info("Step")
	logrus.WithField("IM", p.Id).Info(fmt.Sprintf("%+v %d", state.Sources, len(state.Sources)))
}

func (p *DefaultPartner) WipeOldStates() {

}

func (p *DefaultPartner) initHeightRound(hr HeightRound) *HeightRoundState {
	// first check if there is previous message received
	if _, ok := p.States[hr]; !ok {
		// init one
		p.States[hr] = NewHeightRoundState(p.N)
	}
	return p.States[hr]
}
