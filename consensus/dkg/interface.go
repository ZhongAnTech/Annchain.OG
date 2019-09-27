package dkg

type DkgPeerCommunicatorOutgoing interface {
	Broadcast(msg DkgMessage, peers []PeerInfo)
	Unicast(msg DkgMessage, peer PeerInfo)
}

type DkgPeerCommunicatorIncoming interface {
	GetPipeOut() chan DkgMessage
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
