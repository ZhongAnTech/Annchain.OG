package bft

import (
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
	Myid                   int
	Peers                  []chan BftMessage
	Incoming               chan BftMessage
	ByzantineFeatures      ByzantineFeatures
}

func (d *dummyByzantineBftPeerCommunicator) Broadcast(msg BftMessage, peers []PeerInfo) {
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
		go func(peer PeerInfo) {
			d.Peers[peer.Id] <- msg
		}(peer)
	}
}

func (d *dummyByzantineBftPeerCommunicator) Unicast(msg BftMessage, peer PeerInfo) {
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
		d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyByzantineBftPeerCommunicator) GetIncomingChannel() chan BftMessage {
	return d.Incoming
}

func NewDummyByzantineBftPeerCommunicator(myid int, incoming chan BftMessage, peers []chan BftMessage,
	byzantineFeatures ByzantineFeatures) *dummyByzantineBftPeerCommunicator {
	d := &dummyByzantineBftPeerCommunicator{
		Peers:             peers,
		Myid:              myid,
		Incoming:          incoming,
		ByzantineFeatures: byzantineFeatures,
	}
	return d
}

func (p *dummyByzantineBftPeerCommunicator) doBadThings(msg BftMessage) (updatedMessage BftMessage, toSend bool) {
	updatedMessage = msg
	toSend = true
	switch msg.Type {
	case BftMessageTypeProposal:
		if p.ByzantineFeatures.SilenceProposal {
			toSend = false
		} else if p.ByzantineFeatures.BadProposal {
			v := updatedMessage.Payload.(*MessageProposal)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	case BftMessageTypePreVote:
		if p.ByzantineFeatures.SilencePreVote {
			toSend = false
		} else if p.ByzantineFeatures.BadPreVote {
			v := updatedMessage.Payload.(*MessagePreVote)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}
	case BftMessageTypePreCommit:
		if p.ByzantineFeatures.SilencePreCommit {
			toSend = false
		} else if p.ByzantineFeatures.BadPreCommit {
			v := updatedMessage.Payload.(*MessagePreCommit)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	}
	return
}
