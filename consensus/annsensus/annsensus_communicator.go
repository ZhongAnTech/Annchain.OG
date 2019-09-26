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
type AnnsensusCommunicator struct {
	pipeIn              chan *message.OGMessage
	pipeOut             chan *message.OGMessage
	P2PSender           communicator.P2PSender // upstream message sender
	bftMessageAdapter   BftCommunicatorAdapter
	dkgMessageAdapter   DkgCommunicatorAdapter
	bftPeerCommunicator bft.BftPeerCommunicator
	dkgPeerCommunicator dkg.DkgPeerCommunicator

	quit   chan bool
	quitWg sync.WaitGroup
}

func (r *AnnsensusCommunicator) Run() {
	// keep receiving OG messages and decrypt to incoming channel
	for {
		select {
		case <-r.quit:
			r.quitWg.Done()
			return
		case msg := <-r.pipeIn:
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
	case message.OGMessageType(bft.BftMessageTypeProposal):
		fallthrough
	case message.OGMessageType(bft.BftMessageTypePreVote):
		fallthrough
	case message.OGMessageType(bft.BftMessageTypePreCommit):
		bftMessage, err := ap.bftMessageAdapter.AdaptOgMessage(msg)
		if err != nil {
			logrus.WithError(err).Warn("error on adapting OG message to BFT message")
		}
		// send to bft
		ap.bftPeerCommunicator.GetPipeOut() <- bftMessage
		break
	case message.OGMessageType(dkg.DkgMessageTypeDeal):
		fallthrough
	case message.OGMessageType(dkg.DkgMessageTypeDealResponse):
		fallthrough
	case message.OGMessageType(dkg.DkgMessageTypeSigSets):
		fallthrough
	case message.OGMessageType(dkg.DkgMessageTypeGenesisPublicKey):
		dkgMessage, err := ap.dkgMessageAdapter.AdaptOgMessage(msg)
		if err != nil {
			logrus.WithError(err).Warn("error on adapting OG message to DKG message")
		}
		// send to dkg
		ap.dkgPeerCommunicator.GetPipeOut() <- dkgMessage
		break
	}
}
