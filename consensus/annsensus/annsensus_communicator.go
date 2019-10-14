package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/communicator"
	"github.com/annchain/OG/og/message"
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
func (ap *AnnsensusCommunicator) HandleAnnsensusMessage(msg *message.OGMessage) {
	switch msg.MessageType {
	case message.BinaryMessageType(bft.BftMessageTypeProposal):
		fallthrough
	case message.BinaryMessageType(bft.BftMessageTypePreVote):
		fallthrough
	case message.BinaryMessageType(bft.BftMessageTypePreCommit):
		bftMessage, err := ap.bftMessageAdapter.AdaptOgMessage(msg.Message)
		if err != nil {
			logrus.WithError(err).Warn("error on adapting OG message to BFT message")
		}
		// send to bft
		msgTerm, err := ap.termHolder.GetTermCollection(msg)
		if err != nil {
			logrus.WithError(err).Warn("failed to find appropriate term for msg")
			return
		}
		msgTerm.BftPartner.GetBftPeerCommunicatorIncoming().GetPipeIn() <- &bftMessage
		break
	case message.BinaryMessageType(dkg.DkgMessageTypeDeal):
		fallthrough
	case message.BinaryMessageType(dkg.DkgMessageTypeDealResponse):
		fallthrough
	case message.BinaryMessageType(dkg.DkgMessageTypeSigSets):
		fallthrough
	case message.BinaryMessageType(dkg.DkgMessageTypeGenesisPublicKey):
		dkgMessage, err := ap.dkgMessageAdapter.AdaptOgMessage(msg.Message)
		if err != nil {
			logrus.WithError(err).Warn("error on adapting OG message to DKG message")
		}
		// send to dkg
		msgTerm, err := ap.termHolder.GetTermCollection(msg)
		if err != nil {
			logrus.WithError(err).Warn("failed to find appropriate term for msg")
			return
		}
		msgTerm.DkgPartner.GetDkgPeerCommunicatorIncoming().GetPipeIn() <- &dkgMessage
		break
	}
}

func (ap *AnnsensusCommunicator) BroadcastBft(msg bft.BftMessage, peers []bft.PeerInfo) {
	signed, err := ap.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt bft message to og message")
		return
	}

	for _, peer := range peers {
		ap.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), signed, &peer.PublicKey)
	}
}

func (ap *AnnsensusCommunicator) UnicastBft(msg bft.BftMessage, peer bft.PeerInfo) {
	signed, err := ap.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		logrus.WithError(err).Warn("failed to adapt bft message to og message")
		return
	}
	ap.p2pSender.AnonymousSendMessage(message.BinaryMessageType(msg.Type), signed, &peer.PublicKey)
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
