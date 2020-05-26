package node

type PhysicalCommunicator interface {
	Start()
	Stop()
	GetIncomingChannel() chan *WireMessage
	ClosePeer(id string)
	GetNeighbour(id string) (neighbour *Neighbour, err error)
}
