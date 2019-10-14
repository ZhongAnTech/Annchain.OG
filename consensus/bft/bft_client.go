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

// DefaultBftPartner implements a Tendermint client according to "The latest gossip on BFT consensus"
// DefaultBftPartner is the action performer manipulating the BftStatus.
// It listens to the conditions changed outside (by message or by time) and perform actions.
// Note: Destroy and use a new one upon peers changing.
type DefaultBftPartner struct {
	Id                       int
	BftStatus                *BftStatus
	blockTime                time.Duration
	peerCommunicatorIncoming BftPeerCommunicatorIncoming
	peerCommunicatorOutgoing BftPeerCommunicatorOutgoing
	ProposalGenerator        ProposalGenerator
	ProposalValidator        ProposalValidator
	DecisionMaker            DecisionMaker

	WaiterTimeoutChannel chan *WaiterRequest
	quit                 chan bool
	waiter               *Waiter

	// event listener for a decision once made
	ConsensusReachedListeners []ConsensusReachedListener
	//wg sync.WaitGroup
}

func (p *DefaultBftPartner) GetBftPeerCommunicatorIncoming() BftPeerCommunicatorIncoming {
	return p.peerCommunicatorIncoming
}

func NewDefaultBFTPartner(nbParticipants int, id int, blockTime time.Duration,
	peerCommunicatorIncoming BftPeerCommunicatorIncoming,
	peerCommunicatorOutgoing BftPeerCommunicatorOutgoing,
	proposalGenerator ProposalGenerator,
	proposalValidator ProposalValidator,
	decisionMaker DecisionMaker,
	peerInfo []PeerInfo,
) *DefaultBftPartner {
	if nbParticipants < 2 {
		panic(0)
	}
	p := &DefaultBftPartner{
		Id: id,
		BftStatus: &BftStatus{
			N:      nbParticipants,
			F:      (nbParticipants - 1) / 3,
			Peers:  peerInfo,
			States: make(map[HeightRound]*HeightRoundState),
		},
		blockTime:                blockTime,
		peerCommunicatorIncoming: peerCommunicatorIncoming,
		peerCommunicatorOutgoing: peerCommunicatorOutgoing,
		ProposalGenerator:        proposalGenerator,
		ProposalValidator:        proposalValidator,
		DecisionMaker:            decisionMaker,
		WaiterTimeoutChannel:     make(chan *WaiterRequest, 10),
		quit:                     make(chan bool),
	}

	// TODO: verify if the count is correct
	// p.N == 3 *p.F+1
	if p.BftStatus.N%3 == 1 {
		p.BftStatus.Maj23 = 2*p.BftStatus.F + 1
	} else {
		p.BftStatus.Maj23 = MajorityTwoThird(p.BftStatus.N)
	}

	p.waiter = NewWaiter(p.WaiterTimeoutChannel)

	logrus.WithField("n", p.BftStatus.N).WithField("F", p.BftStatus.F).
		WithField("IM", p.Id).
		WithField("maj23", p.BftStatus.Maj23).Debug("new bft")
	return p
}

// RegisterConsensusReachedListener registers callback for decision made event
// TODO: In the future, protected the array so that it can handle term change
func (p *DefaultBftPartner) RegisterConsensusReachedListener(listener ConsensusReachedListener) {
	p.ConsensusReachedListeners = append(p.ConsensusReachedListeners, listener)
}

func (p *DefaultBftPartner) Start() {
	go p.WaiterLoop()
	go p.EventLoop()
}

func (p *DefaultBftPartner) Stop() {
	//quit channal is used by two or more go routines , use close instead of send values to channel
	close(p.quit)
	close(p.waiter.quit)
	//p.wg.Wait()
	logrus.Info("default partner stopped")
}

func MajorityTwoThird(n int) int {
	return 2*n/3 + 1
}

func (p *DefaultBftPartner) Reset(nbParticipants int, id int) {
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

//func (p *DefaultBftPartner) RestartNewEra() {
//	s := p.BftStatus.States[p.BftStatus.CurrentHR]
//	if s != nil {
//		if s.Decision != nil {
//			//p.BftStatus.States = make(map[p2p_message.HeightRound]*HeightRoundState)
//			p.StartNewEra(p.BftStatus.CurrentHR.Height+1, 0)
//			return
//		}
//		p.StartNewEra(p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round+1)
//		return
//	}
//	//p.BftStatus.States = make(map[p2p_message.HeightRound]*HeightRoundState)
//	p.StartNewEra(p.BftStatus.Curren R.Height, p.BftStatus.CurrentHR.Round)
//	return
//}

func (p *DefaultBftPartner) WaiterLoop() {
	goroutine.New(p.waiter.StartEventLoop)
}

// StartNewEra is called once height or round needs to be changed.
func (p *DefaultBftPartner) StartNewEra(height uint64, round int) {
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
				//	if p.BftStatus.CurrentHR.Height > validCondition.ValidHeight {
				//		//TODO： I received a history message. should be ok?
				//		logrus.WithField("height", p.BftStatus.CurrentHR).WithField("valid height ", validCondition).Warn("height mismatch //TODO")
				//	} else {
				//		//
				//		logrus.WithField("height", p.BftStatus.CurrentHR).WithField("valid height ", validCondition).Debug("height mismatch //TODO")
				//	}
				//
			}
		}
		logrus.WithField("proposal ", proposal).Trace("new proposal")
		// broadcastWaiterTimeoutChannel
		p.Broadcast(BftMessageTypeProposal, p.BftStatus.CurrentHR, proposal, currState.ValidRound)
	} else {
		p.WaitStepTimeout(StepTypePropose, TimeoutPropose, p.BftStatus.CurrentHR, p.OnTimeoutPropose)
	}
}

func (p *DefaultBftPartner) EventLoop() {
	//goroutine.New(p.send)
	//p.wg.Add(1)
	goroutine.New(p.receive)
	//p.wg.Add(1)
}

// receive prevents concurrent state updates by allowing only one channel to be read per loop
// Any action which involves state updates should be in this select clause
func (p *DefaultBftPartner) receive() {
	//defer p.wg.Done()
	timer := time.NewTimer(time.Second * 7)
	pipeOutChannel := p.peerCommunicatorIncoming.GetPipeOut()
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			logrus.Info("got quit msg, bft partner receive routine will stop")
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
			logrus.WithField("IM", p.Id).Debug("Blocked reading incoming bft")
			p.dumpAll("blocked reading incoming")
		case msg := <-pipeOutChannel:
			p.handleMessage(msg)
		}
	}
}

// Proposer returns current round proposer. Now simply round robin
func (p *DefaultBftPartner) Proposer(hr HeightRound) int {
	//return 3
	//return (hr.Height + hr.Round) % p.N
	//maybe overflow
	return (int(hr.Height%uint64(p.BftStatus.N)) + hr.Round%p.BftStatus.N) % p.BftStatus.N
}

// GetValue generates the value requiring consensus
func (p *DefaultBftPartner) GetValue(newBlock bool) (Proposal, ProposalCondition) {
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

	if p.ProposalGenerator != nil {
		pro, validHeight := p.ProposalGenerator.ProduceProposal()
		logrus.WithField("proposal", pro).Debug("proposal gen")
		return pro, validHeight
	}
	v := fmt.Sprintf("■■■%d %d■■■", p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round)
	s := StringProposal(v)
	logrus.WithField("proposal", s).Debug("proposal gen")
	return &s, ProposalCondition{p.BftStatus.CurrentHR.Height}
}

// Broadcast encapsulate messages to all partners
//
func (p *DefaultBftPartner) Broadcast(messageType BftMessageType, hr HeightRound, content Proposal, validRound int) {
	var m BftMessage

	basicInfo := BftBasicInfo{
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
		m = &MessageProposal{
			BftBasicInfo: basicInfo,
			Value:        content,
			ValidRound:   validRound,
		}
	case BftMessageTypePreVote:
		m = &MessagePreVote{
			BftBasicInfo: basicInfo,
			Idv:          idv,
		}
	case BftMessageTypePreCommit:
		m = &MessagePreCommit{
			BftBasicInfo: basicInfo,
			Idv:          idv,
		}
	}
	p.peerCommunicatorOutgoing.Broadcast(m, p.BftStatus.Peers)
}

// OnTimeoutPropose is the callback after staying too long on propose step
func (p *DefaultBftPartner) OnTimeoutPropose(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR && p.BftStatus.States[p.BftStatus.CurrentHR].Step == StepTypePropose {
		p.Broadcast(BftMessageTypePreVote, p.BftStatus.CurrentHR, nil, 0)
		p.changeStep(StepTypePreVote)
	}
}

// OnTimeoutPreVote is the callback after staying too long on prevote step
func (p *DefaultBftPartner) OnTimeoutPreVote(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR && p.BftStatus.States[p.BftStatus.CurrentHR].Step == StepTypePreVote {
		p.Broadcast(BftMessageTypePreCommit, p.BftStatus.CurrentHR, nil, 0)
		p.changeStep(StepTypePreCommit)
	}
}

// OnTimeoutPreCommit is the callback after staying too long on precommit step
func (p *DefaultBftPartner) OnTimeoutPreCommit(context WaiterContext) {
	v := context.(*TendermintContext)
	if v.HeightRound == p.BftStatus.CurrentHR {
		p.StartNewEra(v.HeightRound.Height, v.HeightRound.Round+1)
	}
}

// WaitStepTimeout waits for a centain time if stepType stays too long
func (p *DefaultBftPartner) WaitStepTimeout(stepType StepType, timeout time.Duration, hr HeightRound, timeoutCallback func(WaiterContext)) {
	p.waiter.UpdateRequest(&WaiterRequest{
		WaitTime:        timeout,
		TimeoutCallback: timeoutCallback,
		Context: &TendermintContext{
			HeightRound: hr,
			StepType:    stepType,
		},
	})
}

func (p *DefaultBftPartner) handleMessage(message BftMessage) {
	switch message.GetType() {
	case BftMessageTypeProposal:
		msg, ok := message.(*MessageProposal)
		if !ok {
			logrus.Warn("it claims to be a MessageProposal but the payload does not align")
			return
		}

		if needHandle := p.checkRound(&msg.BftBasicInfo); !needHandle {
			// out-of-date messages, ignore
			break
		}
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.GetType().String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
			"value":  msg.Value,
		}).Debug("In")
		p.handleProposal(msg)
	case BftMessageTypePreVote:
		msg, ok := message.(*MessagePreVote)
		if !ok {
			logrus.Warn("it claims to be a MessagePreVote but the payload does not align")
			return
		}
		if needHandle := p.checkRound(&msg.BftBasicInfo); !needHandle {
			// out-of-date messages, ignore
			break
		}
		p.BftStatus.States[msg.HeightRound].PreVotes[msg.SourceId] = msg
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.GetType().String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreVote(msg)
	case BftMessageTypePreCommit:
		msg, ok := message.(*MessagePreCommit)
		if !ok {
			logrus.Warn("it claims to be a MessagePreCommit but the payload does not align")
			return
		}
		if needHandle := p.checkRound(&msg.BftBasicInfo); !needHandle {
			// out-of-date messages, ignore
			break
		}
		perC := *msg
		p.BftStatus.States[msg.HeightRound].PreCommits[msg.SourceId] = &perC
		logrus.WithFields(logrus.Fields{
			"IM":     p.Id,
			"hr":     p.BftStatus.CurrentHR.String(),
			"type":   message.GetType().String(),
			"from":   msg.SourceId,
			"fromHr": msg.HeightRound.String(),
		}).Debug("In")
		p.handlePreCommit(msg)
	default:
		logrus.WithField("type", message.GetType()).Warn("unknown bft message type")
	}

}
func (p *DefaultBftPartner) handleProposal(proposal *MessageProposal) {
	state, ok := p.BftStatus.States[proposal.HeightRound]
	if !ok {
		logrus.WithField("IM", p.Id).WithField("hr", proposal.HeightRound).Error("proposal height round not in states")
		panic("must exists")
	}
	state.MessageProposal = proposal
	////if this is proposed by me , send precommit
	//if proposal.SourceId == uint16(p.MyIndex)  {
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
func (p *DefaultBftPartner) handlePreVote(vote *MessagePreVote) {
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

func (p *DefaultBftPartner) handlePreCommit(commit *MessagePreCommit) {
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
				// try to validate if we really got a decision
				// This step is usually for value validation
				decision, err := p.DecisionMaker.MakeDecision(state.MessageProposal.Value, state)
				if err != nil {
					logrus.WithError(err).WithField("hr", p.BftStatus.CurrentHR).Warn("validation failed for decision")
					if count == p.BftStatus.N {
						logrus.WithField("hr", p.BftStatus.CurrentHR).Warn("all messages received but not a good decision. Abandom this round")
						p.StartNewEra(p.BftStatus.CurrentHR.Height, p.BftStatus.CurrentHR.Round+1)
					} else {
						logrus.Warn("wait for more correct messages coming")
					}
					return
				}

				// output decision
				state.Decision = decision
				logrus.WithFields(logrus.Fields{
					"IM":    p.Id,
					"hr":    p.BftStatus.CurrentHR.String(),
					"value": state.Decision,
				}).Debug("Decision made")

				//send the decision to upper client to process
				p.notifyDecisionMade(p.BftStatus.CurrentHR, state.Decision)
				// TODO: StartNewEra should be called outside the bft in order to reflect term change.
				// You cannot start new era with height++ by yourself since you are not sure whether you are in the next group
				// Annsensus knows that.
				p.StartNewEra(p.BftStatus.CurrentHR.Height+1, 0)
			}
		}
	}
}

// valid checks proposal validation
func (p *DefaultBftPartner) valid(proposal Proposal) bool {
	err := p.ProposalValidator.ValidateProposal(proposal)
	return err == nil
}

// count votes and commits from others.
func (p *DefaultBftPartner) count(messageType BftMessageType, height uint64, validRound int, valueIdMatchType ValueIdMatchType, valueId *common.Hash) int {
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
func (p *DefaultBftPartner) checkRound(message *BftBasicInfo) (needHandle bool) {
	// check status storage first
	if message.HeightRound.IsAfterOrEqual(p.BftStatus.CurrentHR) {
		_, ok := p.BftStatus.States[message.HeightRound]
		if !ok {
			// create one
			// TODO: verify if someone is generating garbage height
			p.initHeightRound(message.HeightRound)
		}
	} else {
		// this is an old message. just discard it since we don't need to process old messages.
		logrus.WithField("IM", p.Id).WithField("hr", message.HeightRound).Warn("received an old message")
		return false
	}
	// rule line 55
	// slightly changed this so that if there is f+1 newer HeightRound(instead of just round), catch up to this HeightRound
	if message.HeightRound.IsAfter(p.BftStatus.CurrentHR) {
		state, _ := p.BftStatus.States[message.HeightRound]
		//if !ok {
		//	// create one
		//
		//	d, c := p.initHeightRound(message.HeightRound)
		//	state = d
		//	if c != len(p.BftStatus.States) {
		//		panic("number not aligned")
		//	}
		//}
		state.Sources[message.SourceId] = true
		logrus.WithField("IM", p.Id).Tracef("Set source: %d at %s, %+v", message.SourceId, message.HeightRound.String(), state.Sources)
		if _, ok := p.BftStatus.States[message.HeightRound]; !ok {
			panic(fmt.Sprintf("fuck %d %s", p.Id, p.BftStatus.CurrentHR.String()))
		}
		//logrus.WithField("IM", p.MyIndex).Tracef("%d's %s state is %+v, after receiving message %s from %d",
		//	p.MyIndex, p.BftStatus.CurrentHR.String(),
		//	p.BftStatus.States[p.BftStatus.CurrentHR].Sources, message.HeightRound.String(), message.SourceId)

		if len(state.Sources) >= p.BftStatus.F+1 {
			p.dumpAll("New era received")
			p.StartNewEra(message.HeightRound.Height, message.HeightRound.Round)
		}
	}

	return message.HeightRound.IsAfterOrEqual(p.BftStatus.CurrentHR)
}

// changeStep updates the step and then notify the waiter.
func (p *DefaultBftPartner) changeStep(stepType StepType) {
	p.BftStatus.States[p.BftStatus.CurrentHR].Step = stepType
	p.waiter.UpdateContext(&TendermintContext{
		HeightRound: p.BftStatus.CurrentHR,
		StepType:    stepType,
	})
}

// dumpVotes prints all current votes received
func (p *DefaultBftPartner) dumpVotes(votes []*MessagePreVote) string {
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
func (p *DefaultBftPartner) dumpCommits(votes []*MessagePreCommit) string {
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

func (p *DefaultBftPartner) dumpAll(reason string) {
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

func (p *DefaultBftPartner) WipeOldStates() {
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

func (p *DefaultBftPartner) initHeightRound(hr HeightRound) (*HeightRoundState, int) {
	// first check if there is previous message received
	if _, ok := p.BftStatus.States[hr]; !ok {
		// init one
		p.BftStatus.States[hr] = NewHeightRoundState(p.BftStatus.N)
		logrus.WithField("hr", hr).WithField("IM", p.Id).Debug("inited heightround")
	}
	return p.BftStatus.States[hr], len(p.BftStatus.States)
}

type BftStatusReport struct {
	HeightRound HeightRound
	States      HeightRoundStateMap
	Now         time.Time
}

func (p *DefaultBftPartner) Status() interface{} {
	status := BftStatusReport{}
	status.HeightRound = p.BftStatus.CurrentHR
	status.States = p.BftStatus.States
	status.Now = time.Now()
	return &status
}

func (p *DefaultBftPartner) notifyDecisionMade(round HeightRound, decision ConsensusDecision) {
	for _, listener := range p.ConsensusReachedListeners {
		listener.GetConsensusDecisionMadeEventChannel() <- decision
	}
}
