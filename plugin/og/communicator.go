package og

import (
	"github.com/annchain/OG/common/utilfuncs"
	general_communication "github.com/annchain/OG/communication"
	general_message "github.com/annchain/OG/message"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
)

type ProxyOgPeerCommunicator struct {
	OgMessageAdapter OgMessageAdapter // either TrustfulBftAdapter or PlainBftAdapter
	GeneralOutgoing  general_communication.GeneralPeerCommunicatorOutgoing
	pipe             chan *communication.OgMessageEvent
}

func (p *ProxyOgPeerCommunicator) Broadcast(msg message.OgMessage) {
	ogMessage, err := p.OgMessageAdapter.AdaptOgMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")
	p.GeneralOutgoing.Broadcast(ogMessage)
}

func (p *ProxyOgPeerCommunicator) InitDefault() {
	p.pipe = make(chan *communication.OgMessageEvent)
}

func (p ProxyOgPeerCommunicator) Multicast(msg message.OgMessage, peers []communication.OgPeer) {
	ogMessage, err := p.OgMessageAdapter.AdaptOgMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")

	ogPeers := make([]general_message.GeneralPeer, len(peers))
	for i, peer := range peers {
		adaptedValue, err := p.OgMessageAdapter.AdaptOgPeer(&peer)
		utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
		ogPeers[i] = adaptedValue
	}
	p.GeneralOutgoing.Multicast(ogMessage, ogPeers)
}

func (p ProxyOgPeerCommunicator) Unicast(msg message.OgMessage, peer *communication.OgPeer) {
	ogMessage, err := p.OgMessageAdapter.AdaptOgMessage(msg)
	utilfuncs.PanicIfError(err, "Adapter for annsensus should never fail")
	ogPeer, err := p.OgMessageAdapter.AdaptOgPeer(peer)
	utilfuncs.PanicIfError(err, "Adapter for annsensus peer should never fail")
	p.GeneralOutgoing.Unicast(ogMessage, ogPeer)
}

func (p ProxyOgPeerCommunicator) GetPipeIn() chan *communication.OgMessageEvent {
	return p.pipe
}

func (p ProxyOgPeerCommunicator) GetPipeOut() chan *communication.OgMessageEvent {
	return p.pipe
}
