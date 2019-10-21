package communicator

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/types/msg"
	"sync"
)

type OgCommunicator struct {
	p2pSender        P2PSender // upstream message sender
	p2pReceiver      P2PReceiver
	ogMessageAdapter OgMessageAdapter
	quit             chan bool
	quitWg           sync.WaitGroup
}

func (r *OgCommunicator) Run() {
	// keep receiving OG messages and decrypt to incoming channel
	for {
		select {
		case <-r.quit:
			r.quitWg.Done()
			return
		case msg := <-r.p2pReceiver.GetMessageChannel():
			r.HandleOgMessage(msg)
		}
	}
}

func (r *OgCommunicator) HandleOgMessage(message msg.TransportableMessage) {

}

func (r *OgCommunicator) BroadcastOg(msg bft.BftMessage, peers []OgPeerInfo) {

}

func (r *OgCommunicator) UnicastOg(msg bft.BftMessage, peer OgPeerInfo) {

}

func (ap *OgCommunicator) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
}
