package dkg

import "github.com/annchain/OG/og/message"

type DkgPeerCommunicator interface {
	Broadcast(msg DkgMessage, peers []PeerInfo)
	Unicast(msg DkgMessage, peer PeerInfo)
	GetIncomingChannel() chan DkgMessage
	GetReceivingChannel() chan *message.OGMessage
	Run()
}

type DkgGeneratedListener interface {
	GetDkgGeneratedEventChannel() chan bool
}

type DkgOperator interface {
	Start()
	Stop()
	GetPeerCommunicator() DkgPeerCommunicator
	RegisterDkgGeneratedListener(l DkgGeneratedListener)
}
