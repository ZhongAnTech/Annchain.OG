package dkg

type DkgPeerCommunicator interface {
	Broadcast(msg DkgMessage, peers []PeerInfo)
	Unicast(msg DkgMessage, peer PeerInfo)
	GetIncomingChannel() chan DkgMessage
	Run()
}

type DkgGeneratedListener interface {
	GetDkgGeneratedEventChannel() chan bool
}
