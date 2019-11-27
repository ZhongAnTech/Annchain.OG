package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"sync"
)

// AnnsensusPeerCommunicator routes typess and judge which adapter to use and to receive.
// Do IO work only.
type AnnsensusPeerCommunicator struct {
	bftHandler AnnsensusMessageHandler
	dkgHandler AnnsensusMessageHandler
	termHolder HistoricalTermsHolder
	quit       chan bool
	quitWg     sync.WaitGroup
}

func NewAnnsensusCommunicator(
	outgoing AnnsensusPeerCommunicatorOutgoing,
	bftIncoming bft.BftPeerCommunicatorIncoming,
	dkgIncoming dkg.DkgPeerCommunicatorIncoming,
	termHolder HistoricalTermsHolder) *AnnsensusPeerCommunicator {
	return &AnnsensusPeerCommunicator{
		incoming:    incoming,
		outgoing:    outgoing,
		bftIncoming: bftIncoming,
		dkgIncoming: dkgIncoming,
		termHolder:  termHolder,
		quit:        make(chan bool),
		quitWg:      sync.WaitGroup{},
	}
}




func (ap *AnnsensusPeerCommunicator) BroadcastDkg(msg dkg.DkgMessage, peers []dkg.PeerInfo) {
	panic("implement me")
}

func (ap *AnnsensusPeerCommunicator) UnicastDkg(msg dkg.DkgMessage, peer dkg.PeerInfo) {
	panic("implement me")
}

func (ap *AnnsensusPeerCommunicator) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
}
