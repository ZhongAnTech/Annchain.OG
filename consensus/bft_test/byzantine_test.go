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
	Myid                   int
	Peers                  []chan bft.BftMessage
	ReceiverChannel        chan bft.BftMessage
	messageProviderChannel chan bft.BftMessage
	ByzantineFeatures      ByzantineFeatures
}

func (d *dummyByzantineBftPeerCommunicator) GetReceivingChannel() chan bft.BftMessage {
	return d.ReceiverChannel
}

func (d *dummyByzantineBftPeerCommunicator) Run() {
	go func() {
		for {
			v := <-d.ReceiverChannel
			//vv := v.Message.(*bft.BftMessage)
			d.messageProviderChannel <- v
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
			d.Peers[peer.Id] <- msg
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
		d.Peers[peer.Id] <- msg
	}()
}

func (d *dummyByzantineBftPeerCommunicator) GetIncomingChannel() chan bft.BftMessage {
	return d.messageProviderChannel
}

func NewDummyByzantineBftPeerCommunicator(myid int, incoming chan bft.BftMessage, peers []chan bft.BftMessage,
	byzantineFeatures ByzantineFeatures) *dummyByzantineBftPeerCommunicator {
	d := &dummyByzantineBftPeerCommunicator{
		Peers:                  peers,
		Myid:                   myid,
		ReceiverChannel:        incoming,
		messageProviderChannel: make(chan bft.BftMessage),
		ByzantineFeatures:      byzantineFeatures,
	}
	return d
}

func (p *dummyByzantineBftPeerCommunicator) doBadThings(msg bft.BftMessage) (updatedMessage bft.BftMessage, toSend bool) {
	updatedMessage = msg
	toSend = true
	switch msg.Type {
	case bft.BftMessageTypeProposal:
		if p.ByzantineFeatures.SilenceProposal {
			toSend = false
		} else if p.ByzantineFeatures.BadProposal {
			v := updatedMessage.Payload.(*bft.MessageProposal)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	case bft.BftMessageTypePreVote:
		if p.ByzantineFeatures.SilencePreVote {
			toSend = false
		} else if p.ByzantineFeatures.BadPreVote {
			v := updatedMessage.Payload.(*bft.MessagePreVote)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}
	case bft.BftMessageTypePreCommit:
		if p.ByzantineFeatures.SilencePreCommit {
			toSend = false
		} else if p.ByzantineFeatures.BadPreCommit {
			v := updatedMessage.Payload.(*bft.MessagePreCommit)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	}
	return
}
