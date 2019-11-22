package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
)

type AnnsensusPeer struct {
	Id             int
	//PublicKey      crypto.PublicKey `json:"-"`
	//Address        common.Address   `json:"address"`
	//PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

type ProxyBftPeerCommunicator struct {
	bftMessageAdapter     BftMessageAdapter
	annsensusCommunicator *AnnsensusPeerCommunicator
	pipe                  chan bft.BftMessage
}

func NewProxyBftPeerCommunicator(annsensusCommunicator *AnnsensusPeerCommunicator) *ProxyBftPeerCommunicator {
	return &ProxyBftPeerCommunicator{
		annsensusCommunicator: annsensusCommunicator,
		pipe:                  make(chan bft.BftMessage),
	}
}

func (p *ProxyBftPeerCommunicator) Broadcast(msg bft.BftMessage, peers []bft.PeerInfo) {
	annsensusMessage, err := p.bftMessageAdapter.AdaptBftMessage(msg)
	if err != nil {
		panic("adapt should never fail")
	}
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.Broadcast(annsensusMessage, peers)
}

func (p *ProxyBftPeerCommunicator) Unicast(msg bft.BftMessage, peer bft.PeerInfo) {
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.UnicastBft(msg, peer)
}

func (p *ProxyBftPeerCommunicator) GetPipeOut() chan bft.BftMessage {
	// the channel to be consumed by the downstream.
	return p.pipe
}

func (p *ProxyBftPeerCommunicator) GetPipeIn() chan bft.BftMessage {
	// the channel to be fed by other peers
	return p.pipe
}

func (p *ProxyBftPeerCommunicator) Run() {
	// nothing to do
	return
}

type ProxyDkgPeerCommunicator struct {
	annsensusCommunicator *AnnsensusPeerCommunicator
	pipe                  chan dkg.DkgMessage
}

func NewProxyDkgPeerCommunicator(annsensusCommunicator *AnnsensusPeerCommunicator) *ProxyDkgPeerCommunicator {
	return &ProxyDkgPeerCommunicator{
		annsensusCommunicator: annsensusCommunicator,
		pipe:                  make(chan dkg.DkgMessage),
	}
}

func (p *ProxyDkgPeerCommunicator) Broadcast(msg dkg.DkgMessage, peers []dkg.PeerInfo) {
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.BroadcastDkg(msg, peers)
}

func (p *ProxyDkgPeerCommunicator) Unicast(msg dkg.DkgMessage, peer dkg.PeerInfo) {
	// adapt the interface so that the request can be handled by annsensus
	p.annsensusCommunicator.UnicastDkg(msg, peer)
}

func (p *ProxyDkgPeerCommunicator) GetPipeOut() chan dkg.DkgMessage {
	// the channel to be consumed by the downstream.
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) GetPipeIn() chan dkg.DkgMessage {
	// the channel to be fed by other peers
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) Run() {
	// nothing to do
	return
}
