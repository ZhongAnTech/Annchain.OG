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
	"github.com/annchain/OG/common/goroutine"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// BftOperator implements a Tendermint client according to "The latest gossip on BFT consensus"
// BftOperator is the action performer manipulating the BftStatus.
// It listens to the conditions changed outside (by message or by time) and perform actions.
// Note: Destroy and use a new one upon peers changing.
type BftOperator struct {
	Id                int
	BftStatus         BftStatus
	blockTime         time.Duration
	heightProvider    HeightProvider
	PeerCommunicator  BftPeerCommunicator
	proposalGenerator ProposalGenerator

	WaiterTimeoutChannel chan *WaiterRequest
	quit                 chan bool
	waiter               *Waiter
	decisionFunc         func(state *HeightRoundState) error

	testFlag bool

	//wg sync.WaitGroup
}

func NewBFTPartner(nbParticipants int, id int, blockTime time.Duration) *BftOperator {
	if nbParticipants < 2 {
		panic(0)
	}
	p := &BftOperator{
		BftStatus: BftStatus{
			N:      nbParticipants,
			F:      (nbParticipants - 1) / 3,
			States: make(map[HeightRound]*HeightRoundState),
		},
		blockTime:            blockTime,
		Id:                   id,
		WaiterTimeoutChannel: make(chan *WaiterRequest, 10),
		quit:                 make(chan bool),
	}

	// TODO: verify if the count is correct
	// p.N == 3 *p.F+1
	if p.BftStatus.N%3 == 1 {
		p.BftStatus.Maj23 = 2*p.BftStatus.F + 1
	} else {
		p.BftStatus.Maj23 = MajorityTwoThird(p.BftStatus.N)
	}

	p.waiter = NewWaiter(p.WaiterTimeoutChannel)
	p.RegisterDecisionReceiveFunc(deFaultDecisionFunc)

	logrus.WithField("n", p.BftStatus.N).WithField("F", p.BftStatus.F).
		WithField("maj23", p.BftStatus.Maj23).Debug("new bft")
	return p
}

func deFaultDecisionFunc(state *HeightRoundState) error {
	return nil
}

func (p *BftOperator) SetHeightProvider(heightProvider HeightProvider) {
	p.heightProvider = heightProvider
}

func (p *BftOperator) RegisterDecisionReceiveFunc(decisionFunc func(state *HeightRoundState) error) {
	p.decisionFunc = decisionFunc
}

func (p *BftOperator) Stop() {
	//quit channal is used by two or more go routines , use close instead of send values to channel
	close(p.quit)
	close(p.waiter.quit)
	//p.wg.Wait()
	logrus.Info("default partner stopped")
}

func (p *BftOperator) Reset(nbParticipants int, id int) {
	p.BftStatus.N = nbParticipants
	p.BftStatus.F = (nbParticipants - 1) / 3
	p.Id = id
	if p.BftStatus.N%3 == 1 {
		p.BftStatus.Maj23 = 2*p.BftStatus.F + 1
	} else {
		p.BftStatus.Maj23 = MajorityTwoThird(p.BftStatus.N)
	}
	logrus.WithField("maj23", p.BftStatus.Maj23).WithField("f", p.BftStatus.F).
		WithField("nb", p.BftStatus.N).Info("reset bft")
	return
}

func (p *BftOperator) RestartNewEra() {
	s := p.BftStatus.States[p.BftStatus.CurrentHR]
	if s != nil {
		if s.Decision != nil {
			//p.BftStatus.States = make(map[p2p_message.HeightRound]*HeightRoundState)
			p.StartNewEra(p.BftStatus.CurrentHR.Height+1, 0)
			return
		}
		p.StartNewEra(p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round+1)
		return
	}
	//p.BftStatus.States = make(map[p2p_message.HeightRound]*HeightRoundState)
	p.StartNewEra(p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round)
	return
}

func (p *BftOperator) WaiterLoop() {
	goroutine.New(p.waiter.StartEventLoop)
}

// StartNewEra is called once height or round needs to be changed.
func (p *BftOperator) StartNewEra(height uint64, round int) {
	if p.heightProvider != nil {
		ledgerHeight := p.heightProvider.CurrentHeight()
		if ledgerHeight > height {
			height = ledgerHeight
			// TODO: verify if the round needs to reset to 0 once the node is left behind
			round = 0
			logrus.WithField("height ", height).WithField("round", round).Debug("height reset")
		}
	}
	hr := p.BftStatus.CurrentHR
	if height-hr.Height > 1 {
		logrus.WithField("height", height).Warn("height is much higher than current. Indicating packet loss or severe behind.")
	}
	hr.Height = height
	hr.Round = round

	logrus.WithFields(logrus.Fields{
		"IM":        p.Id,
		"currentHR": p.BftStatus.CurrentHR.String(),
		"newHR":     hr.String(),
	}).Debug("Starting new round")

	currState, _ := p.initHeightRound(hr)
	// update partner height
	p.BftStatus.CurrentHR = hr

	p.WipeOldStates()
	p.changeStep(StepTypePropose)

	if p.Id == p.Proposer(p.BftStatus.CurrentHR) {
		logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR.String()).Trace("I'm the proposer")
		var proposal Proposal
		var validCondition ProposalCondition
		if currState.ValidValue != nil {
			logrus.WithField("hr ", hr).Trace("will got valid value")
			proposal = currState.ValidValue
		} else {
			if round == 0 {
				logrus.WithField("hr ", hr).Trace("will got new height value")
				proposal, validCondition = p.GetValue(true)
			} else {
				logrus.WithField("hr ", hr).Trace("will got new round value")
				proposal, validCondition = p.GetValue(false)
			}
			if validCondition.ValidHeight != p.BftStatus.CurrentHR.Height {
				if p.BftStatus.CurrentHR.Height > validCondition.ValidHeight {
					//TODO
					logrus.WithField("height", p.BftStatus.CurrentHR).WithField("valid height ", validCondition).Warn("height mismatch //TODO")
				} else {
					//
					logrus.WithField("height", p.BftStatus.CurrentHR).WithField("valid height ", validCondition).Debug("height mismatch //TODO")
				}

			}
		}
		logrus.WithField("proposal ", proposal).Trace("new proposal")
		// broadcast
		p.Broadcast(BftMessageTypeProposal, p.BftStatus.CurrentHR, proposal, currState.ValidRound)
	} else {
		p.WaitStepTimeout(StepTypePropose, TimeoutPropose, p.BftStatus.CurrentHR, p.OnTimeoutPropose)
	}
}

func (p *BftOperator) EventLoop() {
	//goroutine.New(p.send)
	//p.wg.Add(1)
	goroutine.New(p.receive)
	//p.wg.Add(1)
}

// send is just for outgoing messages. It should not change any state of local tendermint
//func (p *BftOperator) send() {
//	//defer p.wg.Done()
//	timer := time.NewTimer(time.Second * 7)
//	for {
//		timer.Reset(time.Second * 7)
//		select {
//		case <-p.quit:
//			logrus.Info("got quit msg , bft partner send routine will stop")
//			return
//		case <-timer.C:
//			logrus.WithField("IM", p.Id).Warn("Blocked reading outgoing")
//			p.dumpAll("blocked reading outgoing")
//		case msg := <-p.OutgoingMessageChannel:
//			for _, peer := range p.BftStatus.Peers {
//				logrus.WithFields(logrus.Fields{
//					"IM":   p.Id,
//					"hr":   p.BftStatus.CurrentHR.String(),
//					"from": p.Id,
//					"to":   peer.Id,
//					"msg":  msg.String(),
//				}).Debug("Out")
//				//todo may be bug
//				targetPeer := peer
//				goroutine.New(func() {
//					//time.Sleep(time.Duration(300 + rand.Intn(100)) * time.Millisecond)
//					//ffchan.NewTimeoutSenderShort(targetPeer.GetIncomingMessageChannel(), msg, "broadcasting")
//					targetPeer.GetIncomingMessageChannel() <- msg
//				})
//
//			}
//		}
//	}
//}

// receive prevents concurrent state updates by allowing only one channel to be read per loop
// Any action which involves state updates should be in this select clause
func (p *BftOperator) receive() {
	//defer p.wg.Done()
	timer := time.NewTimer(time.Second * 7)
	incomingChannel := p.PeerCommunicator.GetIncomingChannel()
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			logrus.Info("got quit msg , bft partner receive routine will stop")
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
			logrus.WithField("IM", p.Id).Debug("Blocked reading incoming")
			p.dumpAll("blocked reading incoming")
		case msg := <-incomingChannel:
			p.handleMessage(msg)
		}
	}
}

// Proposer returns current round proposer. Now simply round robin
func (p *BftOperator) Proposer(hr HeightRound) int {
	//return 3
	//return (hr.Height + hr.Round) % p.N
	//maybe overflow
	return (int(hr.Height%uint64(p.BftStatus.N)) + hr.Round%p.BftStatus.N) % p.BftStatus.N
}

// GetValue generates the value requiring consensus
func (p *BftOperator) GetValue(newBlock bool) (Proposal, ProposalCondition) {
	//don't sleep for the same height new round
	blockTime := time.After(p.blockTime)
	if newBlock {
		select {
		case <-p.quit:
			logrus.Info("got stop signal")
		case <-blockTime:
			break
		}
		//time.Sleep(p.blockTime)
	}

	if p.proposalGenerator != nil {
		pro, validHeight := p.proposalGenerator.ProduceProposal()
		logrus.WithField("proposal ", pro).Debug("proposal gen ")
		return pro, validHeight
	}
	v := fmt.Sprintf("■■■%d %d■■■", p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round)
	s := StringProposal(v)
	logrus.WithField("proposal ", s).Debug("proposal gen ")
	return &s, ProposalCondition{p.BftStatus.CurrentHR.Height}
}

// Broadcast announce messages to all partners
func (p *BftOperator) Broadcast(messageType BftMessageType, hr HeightRound, content Proposal, validRound int) {
	m := BftMessage{
		Type: messageType,
	}
	basicMessage := BasicMessage{
		HeightRound: hr,
		SourceId:    uint16(p.Id),
	}
	var idv *common.Hash
	if content != nil {
		cIdv := content.GetId()
		if cIdv != nil {
			idv = cIdv
		}

	}
	switch messageType {
	case BftMessageTypeProposal:
		m.Payload = &MessageProposal{
			BasicMessage: basicMessage,
			Value:        content,
			ValidRound:   validRound,
		}
	case BftMessageTypePreVote:
		m.Payload = &MessagePreVote{
			BasicMessage: basicMessage,
			Idv:          idv,
		}
	case BftMessageTypePreCommit:
		m.Payload = &MessagePreCommit{
			BasicMessage: basicMessage,
			Idv:          idv,
		}
	}
	p.PeerCommunicator.Broadcast(m, p.BftStatus.Peers)
}

// OnTimeoutPropose is the callback after staying too long on propose step
func (p *BftOperator) OnTimeoutPropose(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR && p.BftStatus.States[p.BftStatus.CurrentHR].Step == StepTypePropose {
		p.Broadcast(BftMessageTypePreVote, p.BftStatus.CurrentHR, nil, 0)
		p.changeStep(StepTypePreVote)
	}
}

// OnTimeoutPreVote is the callback after staying too long on prevote step
func (p *BftOperator) OnTimeoutPreVote(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR && p.BftStatus.States[p.BftStatus.CurrentHR].Step == StepTypePreVote {
		p.Broadcast(BftMessageTypePreCommit, p.BftStatus.CurrentHR, nil, 0)
		p.changeStep(StepTypePreCommit)
	}
}

// OnTimeoutPreCommit is the callback after staying too long on precommit step
func (p *BftOperator) OnTimeoutPreCommit(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR {
		p.StartNewEra(v.HeightRound.Height, v.HeightRound.Round+1)
	}
}

// WaitStepTimeout waits for a centain time if stepType stays too long
func (p *BftOperator) WaitStepTimeout(stepType StepType, timeout time.Duration, hr HeightRound, timeoutCallback func(WaiterContext)) {
	p.waiter.UpdateRequest(&WaiterRequest{
		WaitTime:        timeout,
		TimeoutCallback: timeoutCallback,
		Context: &TendermintContext{
			HeightRound: hr,
			StepType:    stepType,
		},
	})
}

func (p *BftOperator) handleMessage(message BftMessage) {
	switch message.Type {
	case BftMessageTypeProposal:
		switch message.Payload.(type) {
		case *MessageProposal:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*MessageProposal)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
			break
		}
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
			"value":  msg.Value,
		}).Debug("In")
		p.handleProposal(msg)
	case BftMessageTypePreVote:
		switch message.Payload.(type) {
		case *MessagePreVote:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*MessagePreVote)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
			break
		}
		p.BftStatus.States[msg.HeightRound].PreVotes[msg.SourceId] = msg
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreVote(msg)
	case BftMessageTypePreCommit:
		switch message.Payload.(type) {
		case *MessagePreCommit:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("msg payload error")
		}
		msg := message.Payload.(*MessagePreCommit)
		if needHandle := p.checkRound(&msg.BasicMessage); !needHandle {
			// out-of-date messages, ignore
			break
		}
		perC := *msg
		p.BftStatus.States[msg.HeightRound].PreCommits[msg.SourceId] = &perC
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.Type.String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreCommit(msg)
	}
}
func (p *BftOperator) handleProposal(proposal *MessageProposal) {
	state, ok := p.BftStatus.States[proposal.HeightRound]
	if !ok {
		panic("must exists")
	}
	state.MessageProposal = proposal
	////if this is proposed by me , send precommit
	//if proposal.SourceId == uint16(p.Id)  {
	//	p.Broadcast(BftMessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
	//	p.changeStep(StepTypePreVote)
	//	return
	//}
	// rule line 22
	if state.Step == StepTypePropose {
		if p.valid(proposal.Value) && (state.LockedRound == -1 || state.LockedValue.Equal(proposal.Value)) {
			p.Broadcast(BftMessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
		} else {
			p.Broadcast(BftMessageTypePreVote, proposal.HeightRound, nil, 0)
		}
		p.changeStep(StepTypePreVote)
	}

	// rule line 28
	count := p.count(BftMessageTypePreVote, proposal.HeightRound.Height, proposal.ValidRound, MatchTypeByValue, proposal.Value.GetId())
	if count >= p.BftStatus.Maj23 {
		if state.Step == StepTypePropose && (proposal.ValidRound >= 0 && proposal.ValidRound < p.BftStatus.CurrentHR.Round) {
			if p.valid(proposal.Value) && (state.LockedRound <= proposal.ValidRound || state.LockedValue.Equal(proposal.Value)) {
				p.Broadcast(BftMessageTypePreVote, proposal.HeightRound, proposal.Value, 0)
			} else {
				p.Broadcast(BftMessageTypePreVote, proposal.HeightRound, nil, 0)
			}
			p.changeStep(StepTypePreVote)
		}
	}
}
func (p *BftOperator) handlePreVote(vote *MessagePreVote) {
	// rule line 34
	count := p.count(BftMessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeAny, nil)
	state, ok := p.BftStatus.States[vote.HeightRound]
	if !ok {
		panic("should exists: " + vote.HeightRound.String())
	}
	if count >= p.BftStatus.Maj23 {
		if state.Step == StepTypePreVote && !state.StepTypeEqualPreVoteTriggered {
			logrus.WithField("IM", p.Id).WithField("hr", vote.HeightRound.String()).Debug("prevote counter is more than 2f+1 #1")
			state.StepTypeEqualPreVoteTriggered = true
			p.WaitStepTimeout(StepTypePreVote, TimeoutPreVote, vote.HeightRound, p.OnTimeoutPreVote)
		}
	}
	// rule line 36
	if state.MessageProposal != nil && count >= p.BftStatus.Maj23 {
		if p.valid(state.MessageProposal.Value) && state.Step >= StepTypePreVote && !state.StepTypeEqualOrLargerPreVoteTriggered {
			logrus.WithField("IM", p.Id).WithField("hr", vote.HeightRound.String()).Debug("prevote counter is more than 2f+1 #2")
			state.StepTypeEqualOrLargerPreVoteTriggered = true
			if state.Step == StepTypePreVote {
				state.LockedValue = state.MessageProposal.Value
				state.LockedRound = p.BftStatus.CurrentHR.Round
				p.Broadcast(BftMessageTypePreCommit, vote.HeightRound, state.MessageProposal.Value, 0)
				p.changeStep(StepTypePreCommit)
			}
			state.ValidValue = state.MessageProposal.Value
			state.ValidRound = p.BftStatus.CurrentHR.Round
		}
	}
	// rule line 44
	count = p.count(BftMessageTypePreVote, vote.HeightRound.Height, vote.HeightRound.Round, MatchTypeNil, nil)
	if count >= p.BftStatus.Maj23 && state.Step == StepTypePreVote {
		logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR.String()).Debug("prevote counter is more than 2f+1 #3")
		p.Broadcast(BftMessageTypePreCommit, vote.HeightRound, nil, 0)
		p.changeStep(StepTypePreCommit)
	}

}

func (p *BftOperator) handlePreCommit(commit *MessagePreCommit) {
	// rule line 47
	count := p.count(BftMessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeAny, nil)
	state := p.BftStatus.States[commit.HeightRound]
	if count >= p.BftStatus.Maj23 && !state.StepTypeEqualPreCommitTriggered {
		state.StepTypeEqualPreCommitTriggered = true
		p.WaitStepTimeout(StepTypePreCommit, TimeoutPreCommit, commit.HeightRound, p.OnTimeoutPreCommit)
	}
	// rule line 49
	if state.MessageProposal != nil {
		count = p.count(BftMessageTypePreCommit, commit.HeightRound.Height, commit.HeightRound.Round, MatchTypeByValue, state.MessageProposal.Value.GetId())
		if count >= p.BftStatus.Maj23 {
			if state.Decision == nil {
				// output decision
				state.Decision = state.MessageProposal.Value
				logrus.WithFields(logrus.Fields{
					"IM":    p.Id,
					"hr":    p.BftStatus.CurrentHR.String(),
					"value": state.MessageProposal.Value,
				}).Info("Decision")
				//send the decision to upper client to process
				err := p.decisionFunc(state)
				if err != nil {
					logrus.WithError(err).Warn("commit decision error")
					p.StartNewEra(p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round+1)
				} else {
					p.StartNewEra(p.BftStatus.CurrentHR.Height+1, 0)
				}
			}
		}
	}

}

// valid checks proposal validation
// TODO: inject so that valid will call a function to validate the proposal
func (p *BftOperator) valid(proposal Proposal) bool {
	return true
}

// count votes and commits from others.
func (p *BftOperator) count(messageType BftMessageType, height uint64, validRound int, valueIdMatchType ValueIdMatchType, valueId *common.Hash) int {
	counter := 0

	state, ok := p.BftStatus.States[HeightRound{
		Height: height,
		Round:  validRound,
	}]
	if !ok {
		return 0
	}
	switch messageType {
	case BftMessageTypePreVote:
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
	case BftMessageTypePreCommit:
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
func (p *BftOperator) checkRound(message *BasicMessage) (needHandle bool) {
	// rule line 55
	// slightly changed this so that if there is f+1 newer HeightRound(instead of just round), catch up to this HeightRound
	if message.HeightRound.IsAfter(p.BftStatus.CurrentHR) {
		state, ok := p.BftStatus.States[message.HeightRound]
		if !ok {
			// create one
			// TODO: verify if someone is generating garbage height
			d, c := p.initHeightRound(message.HeightRound)
			state = d
			if c != len(p.BftStatus.States) {
				panic("number not aligned")
			}
		}
		state.Sources[message.SourceId] = true
		logrus.WithField("IM", p.Id).Tracef("Set source: %d at %s, %+v", message.SourceId, message.HeightRound.String(), state.Sources)
		logrus.WithField("IM", p.Id).Tracef("%d's %s state is %+v, after receiving message %s from %d", p.Id, p.BftStatus.CurrentHR.String(), p.BftStatus.States[p.BftStatus.CurrentHR].Sources, message.HeightRound.String(), message.SourceId)

		if len(state.Sources) >= p.BftStatus.F+1 {
			p.dumpAll("New era received")
			p.StartNewEra(message.HeightRound.Height, message.HeightRound.Round)
		}
	}

	return message.HeightRound.IsAfterOrEqual(p.BftStatus.CurrentHR)
}

// changeStep updates the step and then notify the waiter.
func (p *BftOperator) changeStep(stepType StepType) {
	p.BftStatus.States[p.BftStatus.CurrentHR].Step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		HeightRound: p.BftStatus.CurrentHR,
		StepType:    stepType,
	})
}

// dumpVotes prints all current votes received
func (p *BftOperator) dumpVotes(votes []*MessagePreVote) string {
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
func (p *BftOperator) dumpCommits(votes []*MessagePreCommit) string {
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

func (p *BftOperator) dumpAll(reason string) {
	//return
	state := p.BftStatus.States[p.BftStatus.CurrentHR]
	if state == nil {
		logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).WithField("reason", reason).Debug("Dumping nil state")
		return
	}
	logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).WithField("reason", reason).Debug("Dumping")
	logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).WithField("votes", "prevotes").Debug(p.dumpVotes(state.PreVotes))
	logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).WithField("votes", "precommits").Debug(p.dumpCommits(state.PreCommits))
	logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).WithField("step", state.Step.String()).Debug("Step")
	logrus.WithField("IM", p.Id).WithField("hr", p.BftStatus.CurrentHR).Debugf("%+v %d", state.Sources, len(state.Sources))
}

func (p *BftOperator) WipeOldStates() {
	var toRemove []HeightRound
	for hr := range p.BftStatus.States {
		if hr.IsBefore(p.BftStatus.CurrentHR) {
			toRemove = append(toRemove, hr)
		}
	}
	for _, hr := range toRemove {
		delete(p.BftStatus.States, hr)
	}
}

func (p *BftOperator) initHeightRound(hr HeightRound) (*HeightRoundState, int) {
	// first check if there is previous message received
	if _, ok := p.BftStatus.States[hr]; !ok {
		// init one
		p.BftStatus.States[hr] = NewHeightRoundState(p.BftStatus.N)
	}
	return p.BftStatus.States[hr], len(p.BftStatus.States)
}

type BftStatusReport struct {
	HeightRound HeightRound
	States      HeightRoundStateMap
	Now         time.Time
}

func (p *BftOperator) Status() interface{} {
	status := BftStatusReport{}
	status.HeightRound = p.BftStatus.CurrentHR
	status.States = p.BftStatus.States
	status.Now = time.Now()
	return &status
}
