package dkg

type DkgPeerCommunicator interface {
	Broadcast(msg DkgMessage, peers []PeerInfo)
	Unicast(msg DkgMessage, peer PeerInfo)
	GetPipeOut() chan DkgMessage
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
