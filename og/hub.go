package og

import (
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"sync"
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
	peers            *peerSet
	SubProtocols     []p2p.Protocol
	newPeerCh        chan *peer
	maxPeers         int
	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg        sync.WaitGroup
	networkID uint64
}

type HubConfig struct {
	OutgoingBufferSize int
	IncomingBufferSize int
}

func (h *Hub) Init(config *HubConfig) {
	h.outgoing = make(chan *P2PMessage, config.OutgoingBufferSize)
	h.incoming = make(chan *P2PMessage, config.IncomingBufferSize)
	h.quit = make(chan bool)
	h.peers = newPeerSet()
	h.newPeerCh = make(chan *peer)
	h.CallbackRegistry = make(map[MessageType]func(*P2PMessage))
}

func NewHub(config *HubConfig) *Hub {
	h := &Hub{}
	h.Init(config)
	h.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if version < OG32 {
			continue
		}
		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		h.SubProtocols = append(h.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := newPeer(int(version), p, rw)
				select {
				case h.newPeerCh <- peer:
					h.wg.Add(1)
					defer h.wg.Done()
					return h.handle(peer)
				}
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := h.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	return h
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (h *Hub) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if h.peers.Len() >= h.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	log.Debug("og peer connected", "name", p.Name())
	// Execute the Ethereum handshake
	var (
		genesis = types.Tx{} //todo
		head    = types.Hash{}
	)
	if err := p.Handshake(h.networkID, head, genesis.Hash); err != nil {
		log.Debug("OG handshake failed", "err", err)
		return err
	}
	// Register the peer locally
	if err := h.peers.Register(p); err != nil {
		log.Error("og peer registration failed", "err", err)
		return err
	}
	defer h.removePeer(p.id)

	// main loop. handle incoming messages.
	for {
		if err := h.handleMsg(p); err != nil {
			log.Debug("og message handling failed", "err", err)
			return err
		}
	}
}

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *Hub) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	if msg.Code == uint64(StatusMsg) {
		// Handle the message depending on its contentsms

		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
	}
	data, err := ioutil.ReadAll(msg.Payload)
	p2pMsg := P2PMessage{MessageType: MessageType(msg.Code), Message: data}
	p2pMsg.init()
	if p2pMsg.needCheckRepeat {
		p.MarkMessage(p2pMsg.hash)
	}
	h.incoming <- &p2pMsg
	return nil
}

func (h *Hub) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := h.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing og peer", "peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	if err := h.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (h *Hub) Start() {
	go h.loopSend()
	go h.loopReceive()
}

func (h *Hub) Stop() {
	h.quit <- true
	h.quit <- true
	h.peers.Close()
	h.wg.Wait()
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
			log.Info("HubSend reeived quit message. Quitting...")
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
			log.Info("HubReceive received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) SendMessage(messageType MessageType, msg []byte) {
	p2pMsg := P2PMessage{MessageType: messageType, Message: msg}
	if messageType != MessageTypePong && messageType != MessageTypePing {
		p2pMsg.needCheckRepeat = true
		p2pMsg.calculateHash()
	}
	h.outgoing <- &P2PMessage{MessageType: messageType, Message: msg}
}

func (h *Hub) sendMessage(msg *P2PMessage) {
	var peers []*peer
	// choose a peer and then send.
	if msg.needCheckRepeat {
		peers = h.peers.PeersWithoutMsg(msg.hash)
	} else {

		peers = h.peers.Peers()
	}
	for _, peer := range peers {
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
		log.Warnf("Received an unknown message type: %d", msg.MessageType)
	}
}
