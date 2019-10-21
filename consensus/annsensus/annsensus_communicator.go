package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/annchain/OG/types/msg"
	"github.com/sirupsen/logrus"
	"sync"
)

// AnnsensusCommunicator routes ogmessages and judge which adapter to use and to receive.
// Do IO work only.
type AnnsensusCommunicator struct {
	p2pSender         communicator.P2PSender // upstream message sender
	p2pReceiver       communicator.P2PReceiver
	bftMessageAdapter BftMessageAdapter
	dkgMessageAdapter DkgMessageAdapter
	termHolder        TermHolder
	quit              chan bool
	quitWg            sync.WaitGroup
}

func NewAnnsensusCommunicator(
	p2PSender communicator.P2PSender,
	p2pReceiver communicator.P2PReceiver,
	bftMessageAdapter BftMessageAdapter,
	dkgMessageAdapter DkgMessageAdapter,
	termHolder TermHolder) *AnnsensusCommunicator {
	return &AnnsensusCommunicator{
		p2pSender:         p2PSender,
		p2pReceiver:       p2pReceiver,
		bftMessageAdapter: bftMessageAdapter,
		dkgMessageAdapter: dkgMessageAdapter,
		termHolder:        termHolder,
		quit:              nil,
		quitWg:            sync.WaitGroup{},
	}
}

func (r *AnnsensusCommunicator) Run() {
	// keep receiving OG messages and decrypt to incoming channel
	for {
		select {
		case <-r.quit:
			r.quitWg.Done()
			return
		case msg := <-r.p2pReceiver.GetMessageChannel():
			r.HandleAnnsensusMessage(msg)
		}
	}
}

// HandleConsensusMessage is a sub-router for routing consensus message to either bft,dkg or term.
// As part of Annsensus, bft,dkg and term may not be regarded as a separate component of OG.
// Annsensus itself is also a plugin of OG supporting consensus messages.
// Do not block the pipe for any message processing. Router should not be blocked. Use channel.
func (ap *AnnsensusCommunicator) HandleAnnsensusMessage(annsensusMessage msg.TransportableMessage) {
	switch annsensusMessage.GetType() {

	case ogmessage.MessageTypeAnnsensusSigned:
		fallthrough
	case ogmessage.MessageTypeAnnsensusPlain:
		// let bft and dkg handle this message since I don't know the content
		bmsg, err := ap.bftMessageAdapter.AdaptOgMessage(annsensusMessage)
		if err == nil {
			// send to bft
			msgTerm, err := ap.termHolder.GetTermCollection(bmsg)
			if err != nil {
				logrus.WithError(err).Warn("failed to find appropriate term for msg")
				return
			}
			msgTerm.BftPartner.GetBftPeerCommunicatorIncoming().GetPipeIn() <- bmsg
			return
		}

		dmsg, err := ap.dkgMessageAdapter.AdaptOgMessage(annsensusMessage)
		if err == nil {
			// send to dkg
			msgTerm, err := ap.termHolder.GetTermCollection(dmsg)
			if err != nil {
				logrus.WithError(err).Warn("failed to find appropriate term for msg")
				return
			}
			msgTerm.DkgPartner.GetDkgPeerCommunicatorIncoming().GetPipeIn() <- dmsg
			return
		}

		logrus.WithError(err).Warn("error on handling annsensus message")
		return
	default:
		logrus.WithField("msg", annsensusMessage).WithField("IM", ap.termHolder.DebugMyId()).Warn("unsupported annsensus message type")
	}
}

func (ap *AnnsensusCommunicator) BroadcastBft(msg bft.BftMessage, peers []bft.PeerInfo) {
	logrus.WithField("msg", msg).Debug("AnnsensusCommunicator is broadcasting bft message")
	adaptedMessage, err := ap.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt bft message to og message")
		return
	}

	ap.p2pSender.BroadcastMessage(adaptedMessage)
}

func (ap *AnnsensusCommunicator) UnicastBft(msg bft.BftMessage, peer bft.PeerInfo) {
	logrus.WithField("msg", msg).Debug("AnnsensusCommunicator is unicasting bft message")
	adaptedMessage, err := ap.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt bft message to og message")
		return
	}
	ap.p2pSender.AnonymousSendMessage(adaptedMessage, &peer.PublicKey)
}

func (ap *AnnsensusCommunicator) BroadcastDkg(msg dkg.DkgMessage, peers []dkg.PeerInfo) {
	panic("implement me")
}

func (ap *AnnsensusCommunicator) UnicastDkg(msg dkg.DkgMessage, peer dkg.PeerInfo) {
	panic("implement me")
}

func (ap *AnnsensusCommunicator) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
}
