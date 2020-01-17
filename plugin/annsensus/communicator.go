package annsensus

import (
	"github.com/annchain/OG/common/utilfuncs"
	"github.com/annchain/OG/communication"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/message"
)

type ProxyAnnsensusPeerCommunicator struct {
	AnnsensusMessageAdapter AnnsensusMessageAdapter // either TrustfulBftAdapter or PlainBftAdapter
	GeneralOutgoing         communication.GeneralPeerCommunicatorOutgoing
	pipe                    chan *annsensus.AnnsensusMessageEvent
}

func (p *ProxyAnnsensusPeerCommunicator) InitDefault() {
	p.pipe = make(chan *annsensus.AnnsensusMessageEvent)
}

func (p ProxyAnnsensusPeerCommunicator) Broadcast(msg annsensus.AnnsensusMessage, peers []annsensus.AnnsensusPeer) {
	if p.pipe == nil {
		panic("not initialized.")
	}

	ogMessage, err := p.AnnsensusMessageAdapter.AdaptAnnsensusMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")

	ogPeers := make([]message.GeneralPeer, len(peers))
	for i, peer := range peers {
		adaptedValue, err := p.AnnsensusMessageAdapter.AdaptAnnsensusPeer(peer)
		utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
		ogPeers[i] = adaptedValue
	}
	p.GeneralOutgoing.Multicast(ogMessage, ogPeers)
}

func (p ProxyAnnsensusPeerCommunicator) Unicast(msg annsensus.AnnsensusMessage, peer annsensus.AnnsensusPeer) {
	if p.pipe == nil {
		panic("not initialized.")
	}

	ogMessage, err := p.AnnsensusMessageAdapter.AdaptAnnsensusMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")
	ogPeer, err := p.AnnsensusMessageAdapter.AdaptAnnsensusPeer(peer)
	utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
	p.GeneralOutgoing.Unicast(ogMessage, ogPeer)
}

func (p ProxyAnnsensusPeerCommunicator) GetPipeIn() chan *annsensus.AnnsensusMessageEvent {
	if p.pipe == nil {
		panic("not initialized.")
	}
	return p.pipe
}

func (p ProxyAnnsensusPeerCommunicator) GetPipeOut() chan *annsensus.AnnsensusMessageEvent {
	if p.pipe == nil {
		panic("not initialized.")
	}
	return p.pipe
}
