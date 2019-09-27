package annsensus

import "github.com/annchain/OG/consensus/bft"

type ProxyBftPeerCommunicator struct {
	annsensusCommunicator *AnnsensusCommunicator
	pipeOut               chan bft.BftMessage
}

func NewProxyBftPeerCommunicator(annsensusCommunicator *AnnsensusCommunicator) *ProxyBftPeerCommunicator {
	return &ProxyBftPeerCommunicator{
		annsensusCommunicator: annsensusCommunicator,
		pipeOut:               make(chan bft.BftMessage),
	}
}

func (p *ProxyBftPeerCommunicator) Broadcast(msg *bft.BftMessage, peers []bft.PeerInfo) {
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.BroadcastBft(msg, peers)
}

func (p *ProxyBftPeerCommunicator) Unicast(msg *bft.BftMessage, peer bft.PeerInfo) {
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.UnicastBft(msg, peer)
}

func (p *ProxyBftPeerCommunicator) GetPipeOut() chan bft.BftMessage {
	// the channel to be consumed by the downstream.
	return p.pipeOut
}

func (p *ProxyBftPeerCommunicator) Run() {
	// nothing to do
	return
}
