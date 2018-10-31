package og

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/big"

	"github.com/annchain/OG/p2p/discover"
	"github.com/bluele/gcache"
	"sync"
	"time"
)

const (
	softResponseLimit = 4 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)

var errIncompatibleConfig = errors.New("incompatible configuration")

// Hub is the middle layer between p2p and business layer
// When there is a general request coming from the upper layer, Hub will find the appropriate peer to handle.
// When there is a message coming from p2p, Hub will unmarshall this message and give it to message router.
// Hub will also prevent duplicate requests/responses.
// If there is any failure, Hub is NOT responsible for changing a peer and retry. (maybe enhanced in the future.)
// DO NOT INVOLVE ANY BUSINESS LOGICS HERE.
type Hub struct {
	outgoing             chan *P2PMessage
	incoming             chan *P2PMessage
	quit                 chan bool
	CallbackRegistry     map[MessageType]func(*P2PMessage) // All callbacks
	CallbackRegistryOG32 map[MessageType]func(*P2PMessage) // All callbacks of OG32
	StatusDataProvider   NodeStatusDataProvider
	peers                *peerSet
	SubProtocols         []p2p.Protocol

	wg sync.WaitGroup // wait group is used for graceful shutdowns during downloading and processing

	messageCache gcache.Cache // cache for duplicate responses/msg to prevent storm

	maxPeers    int
	newPeerCh   chan *peer
	noMorePeers chan struct{}
	quitSync    chan bool

	// new peer event
	OnNewPeerConnected []chan string
	Downloader         *downloader.Downloader
}

func (h *Hub) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"outgoing":  len(h.outgoing),
		"incoming":  len(h.incoming),
		"newPeerCh": len(h.newPeerCh),
	}
}

type NodeStatusDataProvider interface {
	GetCurrentNodeStatus() StatusData
}

type PeerProvider interface {
	BestPeerInfo() (peerId string, hash types.Hash, seqId uint64, err error)
	GetPeerHead(peerId string) (hash types.Hash, seqId uint64, err error)
}

type HubConfig struct {
	OutgoingBufferSize            int
	IncomingBufferSize            int
	MessageCacheMaxSize           int
	MessageCacheExpirationSeconds int
	MaxPeers                      int
}

func DefaultHubConfig() HubConfig {
	config := HubConfig{
		OutgoingBufferSize:            10,
		IncomingBufferSize:            10,
		MessageCacheMaxSize:           60,
		MessageCacheExpirationSeconds: 3000,
		MaxPeers:                      50,
	}
	return config
}

func (h *Hub) Init(config *HubConfig, dag IDag, txPool ITxPool) {
	h.outgoing = make(chan *P2PMessage, config.OutgoingBufferSize)
	h.incoming = make(chan *P2PMessage, config.IncomingBufferSize)
	h.peers = newPeerSet()
	h.newPeerCh = make(chan *peer)
	h.noMorePeers = make(chan struct{})
	h.quit = make(chan bool)
	h.maxPeers = config.MaxPeers
	h.quitSync = make(chan bool)
	h.messageCache = gcache.New(config.MessageCacheMaxSize).LRU().
		Expiration(time.Second * time.Duration(config.MessageCacheExpirationSeconds)).Build()
	h.CallbackRegistry = make(map[MessageType]func(*P2PMessage))
	h.CallbackRegistryOG32 = make(map[MessageType]func(*P2PMessage))
}

func NewHub(config *HubConfig, dag IDag, txPool ITxPool) *Hub {
	h := &Hub{}
	h.Init(config, dag, txPool)

	h.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		// h.mode == downloader.FastSync &&
		//if version < OG32 {
		//	continue
		//}
		//Compatible; initialise the sub-protocol
		version := version // Closure for the run
		h.SubProtocols = append(h.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := h.newPeer(int(version), p, rw)
				select {
				case <-h.quitSync:
					return p2p.DiscQuitting
				default:
					h.wg.Add(1)
					defer h.wg.Done()
					return h.handle(peer)
				}
			},
			NodeInfo: func() interface{} {
				return h.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := h.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}

	if len(h.SubProtocols) == 0 {
		log.Error(errIncompatibleConfig)
		return nil
	}

	return h
}

func (h *Hub) newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(version, p, rw)
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (h *Hub) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if h.peers.Len() >= h.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	log.WithField("name", p.Name()).WithField("id", p.id).Info("OG peer connected")
	// Execute the og handshake
	statusData := h.StatusDataProvider.GetCurrentNodeStatus()
	//if statusData.CurrentBlock == nil{
	//	panic("Last sequencer is nil")
	//}

	if err := p.Handshake(statusData.NetworkId, statusData.CurrentBlock,
		statusData.CurrentId, statusData.GenesisBlock); err != nil {
		log.WithError(err).WithField("peer ", p.id).Debug("OG handshake failed")
		return err
	}
	// Register the peer locally
	if err := h.peers.Register(p); err != nil {
		log.WithError(err).Error("og peer registration failed")
		return err
	}

	log.Debug("register peer localy")

	defer h.RemovePeer(p.id)
	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := h.Downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}
	//announce new peer
	h.newPeerCh <- p
	// main loop. handle incoming messages.
	for {
		if err := h.handleMsg(p); err != nil {
			log.WithError(err).Debug("og message handling failed")
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
	// Handle the message depending on its contents
	data, err := msg.GetPayLoad()
	p2pMsg := P2PMessage{MessageType: MessageType(msg.Code), Message: data, SourceID: p.id, Version: p.version}
	//log.Debug("start handle p2p messgae ",p2pMsg.MessageType)
	switch {
	case p2pMsg.MessageType == StatusMsg:
		// Handle the message depending on its contentsms

		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
		// Block header query, collect the requested headers and reply
	default:
		p2pMsg.init()
		if p2pMsg.needCheckRepeat {
			p.MarkMessage(p2pMsg.hash)
		}
		h.incoming <- &p2pMsg
		return nil
	}

	return nil
}

func (h *Hub) RemovePeer(id string) {
	// Short circuit if the peer was already removed
	peer := h.peers.Peer(id)
	if peer == nil {
		log.Debug("peer not found id")
		return
	}
	log.WithField("peer", id).Debug("Removing og peer")

	// Unregister the peer from the downloader (should already done) and OG peer set
	h.Downloader.UnregisterPeer(id)
	if err := h.peers.Unregister(id); err != nil {
		log.WithField("peer", "id").WithError(err).
			Error("Peer removal failed")
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (h *Hub) Start() {
	go h.loopSend()
	go h.loopReceive()
	go h.loopNotify()
}

func (h *Hub) Stop() {
	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	//h.noMorePeers <- struct{}{}
	log.Info("quit notifying")
	close(h.quitSync)
	log.Info("quit notified")
	h.peers.Close()
	log.Info("peers closing")
	h.wg.Wait()
	log.Info("peers closed")

	log.Info("hub stopped")
}

func (h *Hub) Name() string {
	return "Hub"
}

func (h *Hub) loopNotify() {
	for {
		select {
		case p := <-h.newPeerCh:
			for _, listener := range h.OnNewPeerConnected {
				listener <- p.id
			}
		case <-h.quit:
			log.Info("Hub-loopNotify received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) loopSend() {
	for {
		select {
		case m := <-h.outgoing:
			// start a new routine in order not to block other communications
			if m.BroadCastToRandom {
				go h.broadcastMessageToRandom(m)
			} else {
				go h.broadcastMessage(m)
			}
		case <-h.quit:
			log.Info("Hub-loopSend received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) loopReceive() {
	for {
		select {
		case m := <-h.incoming:
			// check duplicates
			if _, err := h.messageCache.GetIFPresent(m.hash); err == nil {
				// already there
				msgLog.WithField("from ",m.SourceID).WithField("hash", m.hash).WithField("type", m.MessageType.String()).
					Debug("we have a duplicate message. Discard")
				continue
			}
			h.messageCache.Set(m.hash, nil)
			// start a new routine in order not to block other communications
			go h.receiveMessage(m)
		case <-h.quit:
			log.Info("Hub-loopReceive received quit message. Quitting...")
			return
		}
	}
}

func (h *Hub) BroadcastMessage(messageType MessageType, msg []byte) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg}
	msgOut.init()
	msgLog.WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

func (h *Hub) BroadcastMessageToRandom(messageType MessageType, msg []byte) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg}
	msgOut.init()
	msgOut.BroadCastToRandom = true
	msgLog.WithField("type", messageType).Debug("unicast message")
	h.outgoing <- msgOut
}

func (h *Hub) SendToPeer(peerId string, messageType MessageType, msg types.Message) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.sendRequest(messageType, msg)
}
func (h *Hub) SendBytesToPeer(peerId string, messageType MessageType, msg []byte) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.sendRawMessage(uint64(messageType), msg)
}

// SetPeerHead is just a hack to set the latest seq number known of the peer
// This value ought not to be stored in peer, but an outside map.
// This has nothing related to p2p.
func (h *Hub) SetPeerHead(peerId string, hash types.Hash, number uint64) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	p.SetHead(hash, number)
	return nil
}

func (h *Hub) BestPeerInfo() (peerId string, hash types.Hash, seqId uint64, err error) {
	p := h.peers.BestPeer()
	if p != nil {
		peerId = p.id
		hash, seqId = p.Head()
		return
	}
	err = fmt.Errorf("no best peer")
	return
}

func (h *Hub) GetPeerHead(peerId string) (hash types.Hash, seqId uint64, err error) {
	p := h.peers.Peer(peerId)
	if p != nil {
		hash, seqId = p.Head()
		return
	}
	err = fmt.Errorf("no such peer")
	return
}

func (h *Hub) BestPeerId() (peerId string, err error) {
	p := h.peers.BestPeer()
	if p != nil {
		peerId = p.id
		return
	}
	err = fmt.Errorf("no best peer")
	return
}

func (h *Hub) broadcastMessage(msg *P2PMessage) {
	var peers []*peer
	// choose all  peer and then send.
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
	// h.incoming <- msg
}
func (h *Hub) broadcastMessageToRandom(msg *P2PMessage) {
	peers := h.peers.GetRandomPeers(2)
	// choose random peer and then send.
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
	return
	// DUMMY: Send to me
	// h.incoming <- msg
}

func (h *Hub) receiveMessage(msg *P2PMessage) {
	// route to specific callbacks according to the registry.
	if msg.Version >= OG32 {
		if v, ok := h.CallbackRegistryOG32[msg.MessageType]; ok {
			//log.WithField("from",msg.SourceID).WithField("type", msg.MessageType.String()).Debug("Received a message")
			v(msg)
			return
		}
	}
	if v, ok := h.CallbackRegistry[msg.MessageType]; ok {
		//msgLog.WithField("from",msg.SourceID).WithField("type", msg.MessageType.String()).Debug("Received a message")
		v(msg)
	} else {
		msgLog.WithField("from",msg.SourceID).WithField("type", msg.MessageType).Debug("Received an Unknown message")
	}
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (h *Hub) PeersInfo() []*PeerInfo {
	peers := h.peers.Peers()
	// Gather all the generic and sub-protocol specific infos
	infos := make([]*PeerInfo, 0, len(peers))
	for _, peer := range peers {
		if peer != nil {
			infos = append(infos, peer.Info())
		}
	}
	return infos
}

// NodeInfo represents a short summary of the Ethereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64     `json:"network"`    // Ethereum network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int   `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    types.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	Head       types.Hash `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (h *Hub) NodeInfo() *NodeInfo {
	statusData := h.StatusDataProvider.GetCurrentNodeStatus()
	return &NodeInfo{
		Network: statusData.NetworkId,
		Genesis: statusData.GenesisBlock,
		Head:    statusData.CurrentBlock,
	}
}
