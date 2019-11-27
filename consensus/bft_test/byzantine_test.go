package bft_test

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/sirupsen/logrus"
)

type ByzantineFeatures struct {
	SilenceProposal  bool
	SilencePreVote   bool
	SilencePreCommit bool
	BadProposal      bool
	BadPreVote       bool
	BadPreCommit     bool
}

type dummyByzantineBftPeerCommunicator struct {
	Myid              int
	PeerPipeIns       []chan *bft.BftMessageEvent
	pipeIn            chan *bft.BftMessageEvent
	pipeOut           chan *bft.BftMessageEvent
	ByzantineFeatures ByzantineFeatures
}

func (d *dummyByzantineBftPeerCommunicator) HandleIncomingMessage(msgEvent *bft.BftMessageEvent) {
	d.pipeIn <- msgEvent
}

func (d *dummyByzantineBftPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.pipeIn
			//vv := v.Message.(bft.BftMessage)
			d.pipeOut <- v
		}
	}()
}

func (d *dummyByzantineBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	msg, toSend := d.doBadThings(msg)
	if !toSend {
		// don't send it
		logrus.WithFields(logrus.Fields{
			"IM---BAD": d.Myid,
			"from":     d.Myid,
			"msg":      msg.String(),
		}).Info("Eat broadcast message")
		return
	}
	for _, peer := range peers {
		go func(peer bft.PeerInfo) {
			d.PeerPipeIns[peer.Id] <- &bft.BftMessageEvent{
				Message: msg,
				Peer:    peer,
			}
		}(peer)
	}
}

func (d *dummyByzantineBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	msg, toSend := d.doBadThings(msg)
	if !toSend {
		// don't send it
		logrus.WithFields(logrus.Fields{
			"IM---BAD": d.Myid,
			"from":     d.Myid,
			"to":       peer.Id,
			"msg":      msg.String(),
		}).Info("Eat unicast message")
		return
	}
	go func() {
		d.PeerPipeIns[peer.Id] <- &bft.BftMessageEvent{
			Message: msg,
			Peer:    peer,
		}
	}()
}

func (d *dummyByzantineBftPeerCommunicator) GetPipeIn() chan *bft.BftMessageEvent {
	return d.pipeIn
}

func (d *dummyByzantineBftPeerCommunicator) GetPipeOut() chan *bft.BftMessageEvent {
	return d.pipeOut
}

func NewDummyByzantineBftPeerCommunicator(myid int, incoming chan *bft.BftMessageEvent, peers []chan *bft.BftMessageEvent,
	byzantineFeatures ByzantineFeatures) *dummyByzantineBftPeerCommunicator {
	d := &dummyByzantineBftPeerCommunicator{
		PeerPipeIns:       peers,
		Myid:              myid,
		pipeIn:            incoming,
		pipeOut:           make(chan *bft.BftMessageEvent),
		ByzantineFeatures: byzantineFeatures,
	}
	return d
}

func (p *dummyByzantineBftPeerCommunicator) doBadThings(msg bft.BftMessage) (updatedMessage bft.BftMessage, toSend bool) {
	updatedMessage = msg
	toSend = true
	switch msg.GetType() {
	case bft.BftMessageTypeProposal:
		if p.ByzantineFeatures.SilenceProposal {
			toSend = false
		} else if p.ByzantineFeatures.BadProposal {
			v := updatedMessage.(*bft.BftMessageProposal)
			v.HeightRound.Round++
			updatedMessage = v
		}

	case bft.BftMessageTypePreVote:
		if p.ByzantineFeatures.SilencePreVote {
			toSend = false
		} else if p.ByzantineFeatures.BadPreVote {
			v := updatedMessage.(*bft.BftMessagePreVote)
			v.HeightRound.Round++
			updatedMessage = v
		}
	case bft.BftMessageTypePreCommit:
		if p.ByzantineFeatures.SilencePreCommit {
			toSend = false
		} else if p.ByzantineFeatures.BadPreCommit {
			v := updatedMessage.(*bft.BftMessagePreCommit)
			v.HeightRound.Round++
			updatedMessage = v
		}

	}
	return
}
