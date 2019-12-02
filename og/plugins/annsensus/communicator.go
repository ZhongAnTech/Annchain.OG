package annsensus

import (
	"github.com/annchain/OG/common/utilfuncs"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/og/communication"
)

type ProxyAnnsensusPeerCommunicator struct {
	AnnsensusMessageAdapter AnnsensusMessageAdapter // either TrustfulBftAdapter or PlainBftAdapter
	OgOutgoing              communication.OgPeerCommunicatorOutgoing
	pipe                    chan *annsensus.AnnsensusMessageEvent
}

func (p *ProxyAnnsensusPeerCommunicator) InitDefault() {
	p.pipe = make(chan *annsensus.AnnsensusMessageEvent)
}

func (p ProxyAnnsensusPeerCommunicator) Broadcast(msg annsensus.AnnsensusMessage, peers []annsensus.AnnsensusPeer) {
	ogMessage, err := p.AnnsensusMessageAdapter.AdaptAnnsensusMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")

	ogPeers := make([]communication.OgPeer, len(peers))
	for i, peer := range peers {
		adaptedValue, err := p.AnnsensusMessageAdapter.AdaptAnnsensusPeer(peer)
		utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
		ogPeers[i] = adaptedValue
	}
	p.OgOutgoing.Broadcast(ogMessage, ogPeers)
}

func (p ProxyAnnsensusPeerCommunicator) Unicast(msg annsensus.AnnsensusMessage, peer annsensus.AnnsensusPeer) {
	ogMessage, err := p.AnnsensusMessageAdapter.AdaptAnnsensusMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")
	ogPeer, err := p.AnnsensusMessageAdapter.AdaptAnnsensusPeer(peer)
	utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
	p.OgOutgoing.Unicast(ogMessage, ogPeer)
}

func (p ProxyAnnsensusPeerCommunicator) GetPipeIn() chan *annsensus.AnnsensusMessageEvent {
	return p.pipe
}

func (p ProxyAnnsensusPeerCommunicator) GetPipeOut() chan *annsensus.AnnsensusMessageEvent {
	return p.pipe
}
