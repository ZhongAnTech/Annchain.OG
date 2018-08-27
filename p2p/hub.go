package p2p

import "github.com/sirupsen/logrus"

// Hub is the middle layer between p2p and business layer
// When there is a general request coming from the upper layer, hub will find the appropriate peer to handle.
// Hub will also prevent duplicate requests.
// If there is any failure, Hub is NOT responsible for changing a peer and retry. (maybe enhanced in the future.)
type Hub struct {
	Outgoing         chan *P2PMessage
	Incoming         chan *P2PMessage
	Quit             chan bool
	CallbackRegistry map[uint]func(*P2PMessage) // All callbacks
}

type HubConfig struct {
	OutgoingBufferSize int
	IncomingBufferSize int
}

func (h *Hub) Init(config *HubConfig) {
	h.Outgoing = make(chan *P2PMessage, config.OutgoingBufferSize)
	h.Incoming = make(chan *P2PMessage, config.IncomingBufferSize)
	h.Quit = make(chan bool)
}

func NewHub(config *HubConfig) *Hub {
	h := &Hub{}
	h.Init(config)
	return h
}

func (h *Hub) Start() {
	go h.LoopSend()
	go h.LoopReceive()
}

func (h *Hub) Stop() {
	h.Quit <- true
	h.Quit <- true
}

func (h *Hub) Name() string {
	return "Hub"
}

func (h *Hub) LoopSend() {
	for {
		select {
		case m := <-h.Outgoing:
			// start a new routine in order not to block other communications
			go h.SendMessage(m)
		case <-h.Quit:
			logrus.Info("HubSend received quit message. Quitting...")
			return
		}
	}
}
func (h *Hub) LoopReceive() {
	for {
		select {
		case m := <-h.Incoming:
			// start a new routine in order not to block other communications
			go h.ReceiveMessage(m)
		case <-h.Quit:
			logrus.Info("HubReceive received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) SendMessage(msg *P2PMessage) {
	// choose a peer and then send.
}
func (h *Hub) ReceiveMessage(msg *P2PMessage) {
	// route to specific callbacks according to the registry.
	if v, ok := h.CallbackRegistry[msg.MessageType]; ok {
		v(msg)
	} else {
		logrus.Warnf("Received an unknown message type: %d", msg.MessageType)
	}
}
