package annsensus

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/dkg"
)

type AnnsensusPeer struct {
	Id             int
	PublicKey      crypto.PublicKey `json:"-"`
	Address        common.Address   `json:"address"`
	PublicKeyBytes hexutil.Bytes    `json:"public_key"`
}

type ProxyDkgPeerCommunicator struct {
	annsensusCommunicator *AnnsensusPeerCommunicator
	pipe                  chan *dkg.DkgMessageEvent
}

func NewProxyDkgPeerCommunicator(annsensusCommunicator *AnnsensusPeerCommunicator) *ProxyDkgPeerCommunicator {
	return &ProxyDkgPeerCommunicator{
		annsensusCommunicator: annsensusCommunicator,
		pipe:                  make(chan *dkg.DkgMessageEvent),
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

func (p *ProxyDkgPeerCommunicator) GetPipeOut() chan *dkg.DkgMessageEvent {
	// the channel to be consumed by the downstream.
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) GetPipeIn() chan *dkg.DkgMessageEvent {
	// the channel to be fed by other peers
	return p.pipe
}

func (p *ProxyDkgPeerCommunicator) Run() {
	// nothing to do
	return
}
