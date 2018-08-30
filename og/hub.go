package og

import (
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/types"
)

// Hub is the middle layer between p2p and business layer
// When there is a general request coming from the upper layer, Hub will find the appropriate peer to handle.
// Hub will also prevent duplicate requests.
// If there is any failure, Hub is NOT responsible for changing a peer and retry. (maybe enhanced in the future.)
type Hub struct {
	outgoing         chan *P2PMessage
	incoming         chan *P2PMessage
	quit             chan bool
	CallbackRegistry map[MessageType]func(*P2PMessage) // All callbacks
	peers      *peerSet
}

type HubConfig struct {
	OutgoingBufferSize int
	IncomingBufferSize int
}

func (h *Hub) Init(config *HubConfig) {
	h.outgoing = make(chan *P2PMessage, config.OutgoingBufferSize)
	h.incoming = make(chan *P2PMessage, config.IncomingBufferSize)
	h.quit = make(chan bool)
	h.CallbackRegistry = make(map[MessageType]func(*P2PMessage))
}

func NewHub(config *HubConfig) *Hub {
	h := &Hub{}
	h.Init(config)
	return h
}

func (h *Hub) Start() {
	go h.loopSend()
	go h.loopReceive()
}

func (h *Hub) Stop() {
	h.quit <- true
	h.quit <- true
}

func (h *Hub) Name() string {
	return "Hub"
}

func (h *Hub) loopSend() {
	for {
		select {
		case m := <-h.outgoing:
			// start a new routine in order not to block other communications
			go h.sendMessage(m)
		case <-h.quit:
			logrus.Info("HubSend reeived quit message. Quitting...")
			return
		}
	}
}
func (h *Hub) loopReceive() {
	for {
		select {
		case m := <-h.incoming:
			// start a new routine in order not to block other communications
			go h.receiveMessage(m)
		case <-h.quit:
			logrus.Info("HubReceive received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) SendMessage(messageType MessageType, msgHash types.Hash, msg []byte) {
	h.outgoing <- &P2PMessage{MessageType: messageType,Hash:msgHash, Message: msg}
}

func (h *Hub) sendMessage(msg *P2PMessage) {
	var  peers  []*peer
	// choose a peer and then send.
	switch msg.MessageType {
	case MessageTypeNewTx :
		peers =    h.peers.PeersWithoutTx(msg.Hash)
	default:
		peers = h.peers.Peers()
	}
	for _,peer :=  range peers {
		peer.AsyncSendMessage(msg)
	}
	return
	// DUMMY: Send to me
	//h.incoming <- msg
}

func (h *Hub) receiveMessage(msg *P2PMessage) {
	// route to specific callbacks according to the registry.
	if v, ok := h.CallbackRegistry[msg.MessageType]; ok {
		v(msg)
	} else {
		logrus.Warnf("Received an unknown message type: %d", msg.MessageType)
	}
}
