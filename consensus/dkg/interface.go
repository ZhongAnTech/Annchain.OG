package dkg

type DkgPeerCommunicatorOutgoing interface {
	Broadcast(msg DkgMessage, peers []PeerInfo)
	Unicast(msg DkgMessage, peer PeerInfo)
}

type DkgPeerCommunicatorIncoming interface {
	GetPipeIn() chan *DkgMessageEvent
	GetPipeOut() chan *DkgMessageEvent
}

type DkgGeneratedListener interface {
	GetDkgGeneratedEventChannel() chan bool
}

type DkgPartner interface {
	Start()
	Stop()
	GetDkgPeerCommunicatorIncoming() DkgPeerCommunicatorIncoming
	RegisterDkgGeneratedListener(l DkgGeneratedListener)
}
