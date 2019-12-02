package tendermint

import (
	"github.com/annchain/OG/ffchan"
	"github.com/sirupsen/logrus"
	"time"
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
	*DefaultPartner
	// consider updating resetStatus() if you want to add things here
	ByzantineFeatures ByzantineFeatures
}

func NewByzantinePartner(nbParticipants int, id int, blockTime time.Duration, byzantineFeatures ByzantineFeatures) *ByzantinePartner {
	p := &ByzantinePartner{
		DefaultPartner:    NewPartner(nbParticipants, id, blockTime),
		ByzantineFeatures: byzantineFeatures,
	}
	return p
}

func (p *ByzantinePartner) EventLoop() {
	go p.send()
	go p.receive()
}

// send is just for outgoing messages. It should not change any state of local tendermint
func (p *ByzantinePartner) send() {
	timer := time.NewTimer(time.Second * 7)
	for {
		timer.Reset(time.Second * 7)
		select {
		case <-p.quit:
			break
		case <-timer.C:
			logrus.WithField("IM", p.Id).Warn("Blocked reading outgoing")
			p.dumpAll("blocked reading outgoing(byzantine)")
		case msg := <-p.OutgoingMessageChannel:
			msg, tosend := p.doBadThings(msg)
			if !tosend {
				// don't send it
				logrus.WithFields(logrus.Fields{
					"IM---BAD": p.Id,
					"from":     p.Id,
					"msg":      msg.String(),
				}).Info("Eat message")
				continue
			}
			for _, peer := range p.Peers {
				logrus.WithFields(logrus.Fields{
					"IM---BAD": p.Id,
					"from":     p.Id,
					"to":       peer.GetId(),
					"msg":      msg.String(),
				}).Tracef("PeerOutgoing message")
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
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	case MessageTypePreVote:
		if p.ByzantineFeatures.SilencePreVote {
			toSend = false
		} else if p.ByzantineFeatures.BadPreVote {
			v := updatedMessage.Payload.(MessageCommonVote)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}
	case MessageTypePreCommit:
		if p.ByzantineFeatures.SilencePreCommit {
			toSend = false
		} else if p.ByzantineFeatures.BadPreCommit {
			v := updatedMessage.Payload.(MessageCommonVote)
			v.HeightRound.Round++
			updatedMessage.Payload = v
		}

	}
	return
}
