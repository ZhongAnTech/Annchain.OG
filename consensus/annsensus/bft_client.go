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

package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/goroutine"
	"strings"
	"time"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type BFTPartner interface {
	EventLoop()
	RestartNewAre()
	StartNewEra(height uint64, round int)
	SetPeers(peers []BFTPartner)
	GetIncomingMessageChannel() chan Message
	GetOutgoingMessageChannel() chan Message
	GetWaiterTimeoutChannel() chan *WaiterRequest
	GetId() int
	Proposer(hr types.HeightRound) int
	WaiterLoop()
	GetPeers() (peer []BFTPartner)
	SetProposalFunc(proposalFunc func() (types.Proposal, uint64))
	Stop()
	RegisterDecisionReceiveFunc(decisionFunc func(state *HeightRoundState) error)
	Reset(nbParticipants int, id int)
}

type PartnerBase struct {
	Id                     int
	IncomingMessageChannel chan Message
	OutgoingMessageChannel chan Message
	WaiterTimeoutChannel   chan *WaiterRequest
}

// HeightRoundState is the structure for each Height/Round
// Always keep this state that is higher than current in Partner.States map in order not to miss future things
type HeightRoundState struct {
	MessageProposal                       *types.MessageProposal // the proposal received in this round
	LockedValue                           types.Proposal
	LockedRound                           int
	ValidValue                            types.Proposal
	ValidRound                            int
	Decision                              interface{}               // final decision of mine in this round
	PreVotes                              []*types.MessagePreVote   // other peers' PreVotes
	PreCommits                            []*types.MessagePreCommit // other peers' PreCommits
	Sources                               map[uint16]bool           // for line 55, who send future round so that I may advance?
	StepTypeEqualPreVoteTriggered         bool                      // for line 34, FIRST time trigger
	StepTypeEqualOrLargerPreVoteTriggered bool                      // for line 36, FIRST time trigger
	StepTypeEqualPreCommitTriggered       bool                      // for line 47, FIRST time trigger
	Step                                  StepType                  // current step in this round
}

func NewHeightRoundState(total int) *HeightRoundState {
	return &HeightRoundState{
		LockedRound: -1,
		ValidRound:  -1,
		PreVotes:    make([]*types.MessagePreVote, total),
		PreCommits:  make([]*types.MessagePreCommit, total),
		Sources:     make(map[uint16]bool),
	}
}

func (p *DefaultPartner) GetPeers() []BFTPartner {
	return p.Peers
}

// DefaultPartner implements a Tendermint client according to "The latest gossip on BFT consensus"
// Destroy and use a new one upon peers changing.
type DefaultPartner struct {
	PartnerBase

	CurrentHR types.HeightRound

	blockTime time.Duration
	N         int // total number of participants
	F         int // max number of Byzantines
	Peers     []BFTPartner

	quit         chan bool
	waiter       *Waiter
	proposalFunc func() (types.Proposal, uint64)
	States       map[types.HeightRound]*HeightRoundState // for line 55, round number -> count
	decisionFunc func(state *HeightRoundState) error
	// consider updating resetStatus() if you want to add things here

	testFlag bool
}

func (p *DefaultPartner) GetWaiterTimeoutChannel() chan *WaiterRequest {
	return p.WaiterTimeoutChannel
}

func deFaultDecisionFunc(state *HeightRoundState) error{
	return  nil
}

func (p *DefaultPartner) RegisterDecisionReceiveFunc(decisionFunc func(state *HeightRoundState) error) {
	p.decisionFunc = decisionFunc
}

func (p *DefaultPartner) GetIncomingMessageChannel() chan Message {
	return p.IncomingMessageChannel
}

func (p *DefaultPartner) GetOutgoingMessageChannel() chan Message {
	return p.OutgoingMessageChannel
}

func (P *DefaultPartner) SetProposalFunc(proposalFunc func() (types.Proposal, uint64)) {
	P.proposalFunc = proposalFunc
}

func (p *DefaultPartner) GetId() int {
	return p.Id
}

func (p *DefaultPartner) SetPeers(peers []BFTPartner) {
	p.Peers = peers
}

func (p *DefaultPartner) Stop() {
	p.quit <- true
	p.waiter.quit <- true
	log.Info("default partner stopped")
}

func NewBFTPartner(nbParticipants int, id int, blockTime time.Duration) *DefaultPartner {
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
		States: make(map[types.HeightRound]*HeightRoundState),
	}
	p.waiter = NewWaiter(p.GetWaiterTimeoutChannel())
	p.RegisterDecisionReceiveFunc(deFaultDecisionFunc)
	return p
}

func (p *DefaultPartner) Reset(nbParticipants int, id int) {
	p.N = nbParticipants
	p.F = (nbParticipants - 1) / 3
	p.Id = id
	return

}

func (p *DefaultPartner) RestartNewAre() {
	s := p.States[p.CurrentHR]
	if s != nil {
		if s.Decision != nil {
			//p.States = make(map[types.HeightRound]*HeightRoundState)
			p.StartNewEra(p.CurrentHR.Height+1, 0)
			return
		}
		p.StartNewEra(p.CurrentHR.Height, p.CurrentHR.Round+1)
		return
	}
	//p.States = make(map[types.HeightRound]*HeightRoundState)
	p.StartNewEra(p.CurrentHR.Height, p.CurrentHR.Round)
	return

}

func (p *DefaultPartner) WaiterLoop() {
	goroutine.New(p.waiter.StartEventLoop)
}

// StartNewEra is called once height or round needs to be changed.
func (p *DefaultPartner) StartNewEra(height uint64, round int) {
	hr := p.CurrentHR
	if height-hr.Height > 1 {
		logrus.WithField("height", height).Warn("height is much higher than current. Indicating packet loss or severe behind.")
	}
	hr.Height = height
	hr.Round = round

	log.WithFields(logrus.Fields{
		"IM":        p.Id,
		"currentHR": p.CurrentHR.String(),
		"newHR":     hr.String(),
	}).Debug("Starting new round")

	currState, _ := p.initHeightRound(hr)
	// update partner height
	p.CurrentHR = hr

	p.WipeOldStates()
	p.changeStep(StepTypePropose)

	if p.Id == p.Proposer(p.CurrentHR) {
		logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR.String()).Trace("I'm the proposer")
		var proposal types.Proposal
		var validHeight uint64
		if currState.ValidValue != nil {
			log.WithField("hr ", hr).Trace("will got valid value")
			proposal = currState.ValidValue
		} else {
			if round == 0 {
				log.WithField("hr ", hr).Trace("will got new height value")
				proposal, validHeight = p.GetValue(true)
			} else {
				log.WithField("hr ", hr).Trace("will got new round value")
				proposal, validHeight = p.GetValue(false)
			}
			if validHeight != p.CurrentHR.Height {
				//TODO
				log.WithField("height", p.CurrentHR).WithField("valid height ", validHeight).Warn("height mismatch //TODO")
			}
		}
		log.WithField("proposal ", proposal).Trace("new proposal")
		// broadcast
		p.Broadcast(og.MessageTypeProposal, p.CurrentHR, proposal, currState.ValidRound)
	} else {
		p.WaitStepTimeout(StepTypePropose, TimeoutPropose, p.CurrentHR, p.OnTimeoutPropose)
	}
}

func (p *DefaultPartner) EventLoop() {
	goroutine.New(p.send)
	goroutine.New(p.receive)
}

// send is just for outgoing messages. It should not change any state of local tendermint
func (p *DefaultPartner) send() {
	timer := time.NewTimer(time.Second * 7)
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			log.Info("got quit msg , bft partner send routine will stop")
			return
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
				//todo may be  bug
				targetPeer := peer
				goroutine.New(func() {
					//time.Sleep(time.Duration(300 + rand.Intn(100)) * time.Millisecond)
					//ffchan.NewTimeoutSenderShort(targetPeer.GetIncomingMessageChannel(), msg, "broadcasting")
					targetPeer.GetIncomingMessageChannel() <- msg
				})

			}
		}
	}
}

// receive prevents concurrent state updates by allowing only one channel to be read per loop
// Any action which involves state updates should be in this select clause
func (p *DefaultPartner) receive() {
	timer := time.NewTimer(time.Second * 7)
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			log.Info("got quit msg , bft partner receive routine will stop")
			return
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
func (p *DefaultPartner) Proposer(hr types.HeightRound) int {
	//return 3
	//return (hr.Height + hr.Round) % p.N
	//maybe overflow
	return (int(hr.Height%uint64(p.N)) + hr.Round%p.N) % p.N
}

// GetValue generates the value requiring consensus
func (p *DefaultPartner) GetValue(newBlock bool) (types.Proposal, uint64) {
	//don't sleep for the same height new round
	if newBlock {
		time.Sleep(p.blockTime)
	}
	if p.proposalFunc != nil {
		pro, validHeight := p.proposalFunc()
		logrus.WithField("proposal ", pro).Debug("proposal gen ")
		return pro, validHeight
	}
	v := fmt.Sprintf("■■■%d %d■■■", p.CurrentHR.Height, p.CurrentHR.Round)
	s := types.StringProposal(v)
	logrus.WithField("proposal ", s).Debug("proposal gen ")
	return &s, p.CurrentHR.Height
}

// Broadcast announce messages to all partners
func (p *DefaultPartner) Broadcast(messageType og.MessageType, hr types.HeightRound, content types.Proposal, validRound int) {
	m := Message{
		Type: messageType,
	}
	basicMessage := types.BasicMessage{
		HeightRound: hr,
		SourceId:    uint16(p.Id),
	}
	var idv types.Hash
	if content != nil {
		cIdv := content.GetId()
		if cIdv != nil {
			idv = *cIdv
		}

	}
	switch messageType {
	case og.MessageTypeProposal:
		m.Payload = &types.MessageProposal{
			BasicMessage: basicMessage,
			Value:        content,
			ValidRound:   validRound,
		}
	case og.MessageTypePreVote:
		m.Payload = &types.MessagePreVote{
			BasicMessage: basicMessage,
			Idv:          &idv,
		}
	case og.MessageTypePreCommit:
		m.Payload = &types.MessagePreCommit{
			BasicMessage: basicMessage,
			Idv:          &idv,
		}
	}
	p.OutgoingMessageChannel <- m
	//ffchan.NewTimeoutSenderShort(p.OutgoingMessageChannel, m, "")
}

// OnTimeoutPropose is the callback after staying too long on propose step
func (p *DefaultPartner) OnTimeoutPropose(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR && p.States[p.CurrentHR].Step == StepTypePropose {
		p.Broadcast(og.MessageTypePreVote, p.CurrentHR, nil, 0)
		p.changeStep(StepTypePreVote)
	}
}

// OnTimeoutPreVote is the callback after staying too long on prevote step
func (p *DefaultPartner) OnTimeoutPreVote(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR && p.States[p.CurrentHR].Step == StepTypePreVote {
		p.Broadcast(og.MessageTypePreCommit, p.CurrentHR, nil, 0)
		p.changeStep(StepTypePreCommit)
	}
}

// OnTimeoutPreCommit is the callback after staying too long on precommit step
func (p *DefaultPartner) OnTimeoutPreCommit(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.CurrentHR {
		p.StartNewEra(v.HeightRound.Height, v.HeightRound.Round+1)
	}
}

// WaitStepTimeout waits for a centain time if stepType stays too long
func (p *DefaultPartner) WaitStepTimeout(stepType StepType, timeout time.Duration, hr types.HeightRound, timeoutCallback func(WaiterContext)) {
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
	switch message.Type {
	case og.MessageTypeProposal:
		switch message.Payload.(type) {
		case *types.MessageProposal:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*types.MessageProposal)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
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
		p.handleProposal(msg)
	case og.MessageTypePreVote:
		switch message.Payload.(type) {
		case *types.MessagePreVote:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*types.MessagePreVote)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
			break
		}
		p.States[msg.HeightRound].PreVotes[msg.SourceId] = msg
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreVote(msg)
	case og.MessageTypePreCommit:
		switch message.Payload.(type) {
		case *types.MessagePreCommit:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*types.MessagePreCommit)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
			break
		}
		perC := *msg
		p.States[msg.HeightRound].PreCommits[msg.SourceId] = &perC
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreCommit(msg)
	}
}
func (p *DefaultPartner) handleProposal(proposal *types.MessageProposal) {
	state, ok := p.States[proposal.HeightRound]
	if !ok {
		panic("must exists")
	}
	state.MessageProposal = proposal
	////if this is proposed by me , send precommit
	//if proposal.SourceId == uint16(p.Id)  {
	//	p.Broadcast(og.MessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
	//	p.changeStep(StepTypePreVote)
	//	return
	//}
	// rule line 22
	if state.Step == StepTypePropose {
		if p.valid(proposal.Value) && (state.LockedRound == -1 || state.LockedValue.Equal(proposal.Value)) {
			p.Broadcast(og.MessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
		} else {
			p.Broadcast(og.MessageTypePreVote, proposal.HeightRound, nil, 0)
		}
		p.changeStep(StepTypePreVote)
	}

	// rule line 28
	count := p.count(og.MessageTypePreVote, proposal.HeightRound.Height, proposal.ValidRound, MatchTypeByValue, proposal.Value.GetId())
	if count >= 2*p.F+1 {
		if state.Step == StepTypePropose && (proposal.ValidRound >= 0 && proposal.ValidRound < p.CurrentHR.Round) {
			if p.valid(proposal.Value) && (state.LockedRound <= proposal.ValidRound || state.LockedValue.Equal(proposal.Value)) {
				p.Broadcast(og.MessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
			} else {
				p.Broadcast(og.MessageTypePreVote, proposal.HeightRound, nil, 0)
			}
			p.changeStep(StepTypePreVote)
		}
	}
}
func (p *DefaultPartner) handlePreVote(vote *types.MessagePreVote) {
	// rule line 34
	count := p.count(og.MessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeAny, nil)
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
				p.Broadcast(og.MessageTypePreCommit, vote.HeightRound, state.MessageProposal.Value, 0)
				p.changeStep(StepTypePreCommit)
			}
			state.ValidValue = state.MessageProposal.Value
			state.ValidRound = p.CurrentHR.Round
		}
	}
	// rule line 44
	count = p.count(og.MessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeNil, nil)
	if count >= 2*p.F+1 && state.Step == StepTypePreVote {
		logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR.String()).Debug("prevote counter is more than 2f+1 #3")
		p.Broadcast(og.MessageTypePreCommit, vote.HeightRound, nil, 0)
		p.changeStep(StepTypePreCommit)
	}

}

func (p *DefaultPartner) handlePreCommit(commit *types.MessagePreCommit) {
	// rule line 47
	count := p.count(og.MessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeAny, nil)
	state := p.States[commit.HeightRound]
	if count >= 2*p.F+1 && !state.StepTypeEqualPreCommitTriggered {
		state.StepTypeEqualPreCommitTriggered = true
		p.WaitStepTimeout(StepTypePreCommit, TimeoutPreCommit, commit.HeightRound, p.OnTimeoutPreCommit)
	}
	// rule line 49
	if state.MessageProposal != nil {
		count = p.count(og.MessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeByValue, state.MessageProposal.Value.GetId())
		if count >= 2*p.F+1 {
			if state.Decision == nil {
				// output decision
				state.Decision = state.MessageProposal.Value
				logrus.WithFields(logrus.Fields{
					"IM":    p.Id,
					"hr":    p.CurrentHR.String(),
					"value": state.MessageProposal.Value,
				}).Info("Decision")
				//send the decision to upper client to process
				err := p.decisionFunc(state)
				if err != nil {
					log.WithError(err).Warn("commit decision error")
					p.StartNewEra(p.CurrentHR.Height, p.CurrentHR.Round+1)
				} else {
					p.StartNewEra(p.CurrentHR.Height+1, 0)
				}
			}
		}
	}

}

// valid checks proposal validation
// TODO: inject so that valid will call a function to validate the proposal
func (p *DefaultPartner) valid(proposal types.Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *DefaultPartner) count(messageType og.MessageType, height uint64, validRound int, valueIdMatchType ValueIdMatchType, valueId *types.Hash) int {
	counter := 0

	state, ok := p.States[types.HeightRound{
		Height: height,
		Round:  validRound,
	}]
	if !ok {
		return 0
	}
	switch messageType {
	case og.MessageTypePreVote:
		target := state.PreVotes
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
					counter++
				}
			case MatchTypeNil:
				if m.Idv == nil {
					counter++
				}
			case MatchTypeAny:
				counter++
			}
		}
	case og.MessageTypePreCommit:
		target := state.PreCommits
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
				if m.Idv == nil {
					if valueId == nil {
						counter++
					}
				} else if valueId != nil && *valueId == *m.Idv {
					counter++
				}
			case MatchTypeNil:
				if m.Idv == nil {
					counter++
				}
			case MatchTypeAny:
				counter++
			}
		}
	default:
		//panic("not implemented")
	}
	logrus.WithField("IM", p.Id).
		Debugf("Counting: [%d] %s H:%d VR:%d MT:%d", counter, messageType.String(), height, validRound, valueIdMatchType)
	return counter
}

// checkRound will init all data structure this message needs.
// It also check if the message is out of date, or advanced too much
func (p *DefaultPartner) checkRound(message *types.BasicMessage) (needHandle bool) {
	// rule line 55
	// slightly changed this so that if there is f+1 newer HeightRound(instead of just round), catch up to this HeightRound
	if message.HeightRound.IsAfter(p.CurrentHR) {
		state, ok := p.States[message.HeightRound]
		if !ok {
			// create one
			// TODO: verify if someone is generating garbage height
			d, c := p.initHeightRound(message.HeightRound)
			state = d
			if c != len(p.States) {
				panic("number not aligned")
			}
		}
		state.Sources[message.SourceId] = true
		logrus.WithField("IM", p.Id).Tracef("Set source: %d at %s, %+v", message.SourceId, message.HeightRound.String(), state.Sources)
		logrus.WithField("IM", p.Id).Tracef("%d's %s state is %+v, after receiving message %s from %d", p.Id, p.CurrentHR.String(), p.States[p.CurrentHR].Sources, message.HeightRound.String(), message.SourceId)

		if len(state.Sources) >= p.F+1 {
			p.dumpAll("New era received")
			p.StartNewEra(message.HeightRound.Height, message.HeightRound.Round)
		}
	}

	return message.HeightRound.IsAfterOrEqual(p.CurrentHR)
}

// changeStep updates the step and then notify the waiter.
func (p *DefaultPartner) changeStep(stepType StepType) {
	p.States[p.CurrentHR].Step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		HeightRound: p.CurrentHR,
		StepType:    stepType,
	})
}

// dumpVotes prints all current votes received
func (p *DefaultPartner) dumpVotes(votes []*types.MessagePreVote) string {
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

// dumpVotes prints all current votes received
func (p *DefaultPartner) dumpCommits(votes []*types.MessagePreCommit) string {
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
	if state == nil {
		logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).WithField("reason", reason).Debug("Dumping nil state")
		return
	}
	logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).WithField("reason", reason).Debug("Dumping")
	logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).WithField("votes", "prevotes").Debug(p.dumpVotes(state.PreVotes))
	logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).WithField("votes", "precommits").Debug(p.dumpCommits(state.PreCommits))
	logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).WithField("step", state.Step.String()).Debug("Step")
	logrus.WithField("IM", p.Id).WithField("hr", p.CurrentHR).Debugf("%+v %d", state.Sources, len(state.Sources))
}

func (p *DefaultPartner) WipeOldStates() {
	var toRemove []types.HeightRound
	for hr := range p.States {
		if hr.IsBefore(p.CurrentHR) {
			toRemove = append(toRemove, hr)
		}
	}
	for _, hr := range toRemove {
		delete(p.States, hr)
	}
}

func (p *DefaultPartner) initHeightRound(hr types.HeightRound) (*HeightRoundState, int) {
	// first check if there is previous message received
	if _, ok := p.States[hr]; !ok {
		// init one
		p.States[hr] = NewHeightRoundState(p.N)
	}
	return p.States[hr], len(p.States)
}
