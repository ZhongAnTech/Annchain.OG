package og

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/og/fetcher"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/big"

	"github.com/annchain/OG/p2p/enode"
	"github.com/bluele/gcache"
	"sync"
	"time"
)

const (
	softResponseLimit = 4 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize          = 4096
	DuplicateMsgPeerNum = 5
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
	Fetcher            *fetcher.Fetcher

	NodeInfo    func() *p2p.NodeInfo
	IsKnownHash func(hash types.Hash) bool
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

func (h *Hub) Init(config *HubConfig) {
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

func NewHub(config *HubConfig) *Hub {
	h := &Hub{}
	h.Init(config)

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
			Length:  p2p.MsgCodeType(ProtocolLengths[i]),
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
				return h.NodeStatus()
			},
			PeerInfo: func(id enode.ID) interface{} {
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
	p2pMsg := P2PMessage{MessageType: MessageType(msg.Code), data: data, SourceID: p.id, Version: p.version}
	//log.Debug("start handle p2p messgae ",p2pMsg.MessageType)
	switch {
	case p2pMsg.MessageType == StatusMsg:
		// Handle the message depending on its contentsms

		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
		// Block header query, collect the requested headers and reply
	case p2pMsg.MessageType == MessageTypeDuplicate:
		msgLog.WithField("got msg", MessageTypeDuplicate).WithField("peer ", p.String()).Info("set path to false")
		if out, _ := p.CheckPath(); out {
			p.SetOutPath(false)
		}

		return nil

	default:
		duplicate, err := h.checkMsg(&p2pMsg)
		if duplicate {
			out, in := p.CheckPath()
			log.WithField("type", p2pMsg.MessageType).WithField("msg", p2pMsg.Message.String()).WithField(
				"hash", p2pMsg.hash).WithField("from ", p.String()).WithField("out ", out).WithField("in ", in).Debug("duplicate msg ,discard")
			if p2pMsg.MessageType == MessageTypeNewTx || p2pMsg.MessageType == MessageTypeNewSequencer {
				if outNum, inNum := h.peers.ValidPathNum(); inNum <= 1 {
					log.WithField("outNum ", outNum).WithField("inNum", inNum).Debug("not enough valid path")
					//return nil
				}
				var  dup types.MessageDuplicate
				p.SetInPath(false)
				return h.SendToPeer(p.id, MessageTypeDuplicate, &dup)
			}
			return nil
		}
		if err != nil {
			log.WithField("type ", p2pMsg.MessageType).WithError(err).Warn("handle msg error")
			return err
		}
		p.MarkMessage(p2pMsg.hash)
		msgLog.WithField("type", p2pMsg.MessageType.String()).WithField("from", p.String()).WithField(
			"Message", p2pMsg.Message.String()).WithField("len ", len(p2pMsg.data)).Debug("received a message")
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
	h.Fetcher.Start()
	go h.loopSend()
	go h.loopReceive()
	go h.loopNotify()
}

func (h *Hub) Stop() {
	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	h.noMorePeers <- struct{}{}
	log.Info("quit notifying")
	close(h.quitSync)
	log.Info("quit notified")
	h.peers.Close()
	log.Info("peers closing")
	h.wg.Wait()
	log.Info("peers closed")
	h.Fetcher.Stop()
	log.Info("fetcher stopped")
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
			switch m.sendingType {
			case sendingTypeBroacast:
				go h.broadcastMessage(m)
			case sendingTypeMulticast:
				go h.multicastMessage(m)
			case sendingTypeMulticastToSource:
				h.multicastMessageToSource(m)
			case sendingTypeBroacastWithFilter:
				go h.broadcastMessage(m)
			case sendingTypeBroacastWithLink:
				go h.broadcastMessageWithLink(m)

			default:
				log.WithField("type ", m.sendingType).Error("unknown sending  type")
				panic(m)
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
			// start a new routine in order not to block other communications
			go h.receiveMessage(m)
		case <-h.quit:
			log.Info("Hub-loopReceive received quit message. Quitting...")
			return
		}
	}
}

//MulticastToSource  multicast msg to source , for example , send tx request to the peer which hash the tx
func (h *Hub) MulticastToSource(messageType MessageType, msg types.Message, sourceMsgHash *types.Hash) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg, sendingType: sendingTypeMulticastToSource, SourceHash: sourceMsgHash}
	_, err := h.checkMsg(msgOut)
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("multicast msg to source")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessage(messageType MessageType, msg types.Message) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg, sendingType: sendingTypeBroacast}
	_, err := h.checkMsg(msgOut)
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessageWithLink(messageType MessageType, msg types.Message) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg, sendingType: sendingTypeBroacastWithLink}
	_, err := h.checkMsg(msgOut)
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessageWithFilter(messageType MessageType, msg types.Message) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg, sendingType: sendingTypeBroacastWithFilter}
	_, err := h.checkMsg(msgOut)
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//MulticastMessage multicast message to some peer
func (h *Hub) MulticastMessage(messageType MessageType, msg types.Message) {
	msgOut := &P2PMessage{MessageType: messageType, Message: msg, sendingType: sendingTypeMulticast}
	_, err := h.checkMsg(msgOut)
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
	}
	msgLog.WithField("size", len(msgOut.data)).WithField("type", messageType).Trace("multicast message")
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
	return p.sendRawMessage(messageType, msg)
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

func (h *Hub) checkMsg(m *P2PMessage) (duplicate bool, err error) {
	//for incoming msg
	if m.SourceID != "" {
		err := m.GetMessage()
		if err != nil {
			return false, err
		}
		_, err = m.Message.UnmarshalMsg(m.data)
		if err != nil {
			return false, err
		}
		m.calculateHash()
		if h.cacheMessage(m) {
			return true, nil
		}

	} else {
		//outgoing msg
		data, err := m.Message.MarshalMsg(nil)
		if err != nil {
			return false, err
		}
		m.data = data
		m.calculateHash()
	}
	return false, nil
}

//broadcastMessage
func (h *Hub) broadcastMessage(msg *P2PMessage) {
	var peers []*peer
	// choose all  peer and then send.
	peers = h.peers.PeersWithoutMsg(msg.hash)
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
	return
}

func (h *Hub) broadcastMessageWithLink(msg *P2PMessage) {
	var peers []*peer
	// choose all  peer and then send.
	hash := msg.hash
	c := types.MessageControl{Hash: &hash}
	var pMsg = &P2PMessage{MessageType: MessageTypeControl, Message: &c}
	h.checkMsg(pMsg)
	peers = h.peers.PeersWithoutMsg(msg.hash)

	for _, peer := range peers {
		if peer.inBound || len(peers) ==1{
		//if out, _ := peer.CheckPath(); out {
			peer.AsyncSendMessage(msg)
		} else {
			continue
			if !peer.knownMsg.Contains(pMsg.hash) {
				msgLog.WithField("hash ", hash).Debug("send MessageTypeControl")
				peer.AsyncSendMessage(pMsg)
			} else {
				msgLog.WithField("hash ", hash).Debug("contains MessageTypeControl")
			}
		}
	}
	return
}

/*
func (h *Hub) broadcastMessageWithFilter(msg *P2PMessage) {
	newSeq := msg.Message.(*types.MessageNewSequencer)
	if newSeq.Filter == nil {
		newSeq.Filter = types.NewDefaultBloomFilter()
	} else if len(newSeq.Filter.Data) != 0 {
		err := newSeq.Filter.Decode()
		if err != nil {
			msgLog.WithError(err).Warn("encode bloom filter error")
			return
		}
	}
	var allpeers []*peer
	var peers []*peer
	allpeers = h.peers.PeersWithoutMsg(msg.hash)
	for _, peer := range allpeers {
		ok, _ := newSeq.Filter.LookUpItem(peer.ID().Bytes())
		if ok {
			msgLog.WithField("id ", peer.id).Debug("filtered ,don't send")
		} else {
			newSeq.Filter.AddItem(peer.ID().Bytes())
			peers = append(peers, peer)
			msgLog.WithField("id ", peer.id).Debug("not filtered , send")
		}
	}
	newSeq.Filter.Encode()
	msg.Message = newSeq
	msg.data, _ = newSeq.MarshalMsg(nil)
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
}

*/

//multicastMessage
func (h *Hub) multicastMessage(msg *P2PMessage) error {
	peers := h.peers.GetRandomPeers(2)
	// choose random peer and then send.
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
	return nil
}

//multicastMessageToSource
func (h *Hub) multicastMessageToSource(msg *P2PMessage) error {
	if msg.SourceHash == nil {
		msgLog.Warn("source msg hash is nil , multuicast to random ")
		return h.multicastMessage(msg)
	}
	ids := h.getMsgFromCache(*msg.SourceHash)
	//send to 2 peer , considering if one peer disconnect,
	peers := h.peers.GetPeers(ids, 2)
	if len(peers) == 0 {
		msgLog.WithField("type ", msg.MessageType).WithField("peeers id ", ids).Warn(
			"not found source peers, multicast to random")
		return h.multicastMessage(msg)
	}
	// choose random peer and then send.
	for _, peer := range peers {
		if peer == nil {
			continue
		}
		peer.AsyncSendMessage(msg)
	}
	return nil
}

//cacheMessge save msg to cache
func (h *Hub) cacheMessage(m *P2PMessage) (exists bool) {
	var peers []string
	if a, err := h.messageCache.GetIFPresent(m.hash); err == nil {
		// already there
		exists = true
		//var peers []string
		peers = a.([]string)
		msgLog.WithField("from ", m.SourceID).WithField("hash", m.hash).WithField("peers", peers).WithField("type", m.MessageType.String()).
			Trace("we have a duplicate message. Discard")
		if len(peers) == 0 {
			msgLog.Error("peers is nil")
		} else if len(peers) >= DuplicateMsgPeerNum {
			return
		}
	}
	peers = append(peers, m.SourceID)
	h.messageCache.Set(m.hash, peers)
	return exists
}

//getMsgFromCache
func (h *Hub) getMsgFromCache(hash types.Hash) []string {
	if a, err := h.messageCache.GetIFPresent(hash); err == nil {
		var peers []string
		peers = a.([]string)
		msgLog.WithField("peers ", peers).Trace("get peers from cache ")
		return peers
	}
	return nil
}

func (h *Hub) HandleControlMsg(msg *P2PMessage) {
	req := msg.Message.(*types.MessageControl)
	if req.Hash == nil {
		msgLog.WithError(fmt.Errorf("miss hash")).Debug("control msg request err")
		return
	}
	hash := *req.Hash
	source := msg.SourceID
	if _, err := h.messageCache.GetIFPresent(hash); err == nil {
		msgLog.WithField("hash", hash).Debug("got msg already")
		return
	}
	var i int
	for {
		select {
		case <-time.After(5 * time.Millisecond):
			i++
			if _, err := h.messageCache.GetIFPresent(hash); err == nil {
				msgLog.WithField("duration", i*5).WithField("hash", hash).Debug()
				return

			}
			if h.IsKnownHash(hash) {
				return
			}

			if i > 10 {
				getMsg := types.MessageGetMsg{Hash: &hash}
				h.SendToPeer(source, MessageTypeGetMsg, &getMsg)
				p := h.peers.Peer(source)
				if p != nil {
					if _, in := p.CheckPath(); in {
						p.SetInPath(true)
					}
				}
				return
			}
		}

	}

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

	if msg.MessageType == MessageTypeControl {
		go h.HandleControlMsg(msg)
		return
	}
	if msg.MessageType == MessageTypeGetMsg {
		peer := h.peers.Peer(msg.SourceID)
		if peer != nil {
			msgLog.WithField("msg", msg.Message.String()).WithField("peer ", peer.String()).Info("set path to true")
			peer.SetOutPath(true)
		}
	}

	if v, ok := h.CallbackRegistry[msg.MessageType]; ok {
		msgLog.WithField("type", msg.MessageType).Debug("handle")
		v(msg)
	} else {
		msgLog.WithField("from", msg.SourceID).WithField("type", msg.MessageType).Debug("Received an Unknown message")
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
type NodeStatus struct {
	Network    uint64     `json:"network"`    // Ethereum network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int   `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    types.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	Head       types.Hash `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (h *Hub) NodeStatus() *NodeStatus {
	statusData := h.StatusDataProvider.GetCurrentNodeStatus()
	return &NodeStatus{
		Network: statusData.NetworkId,
		Genesis: statusData.GenesisBlock,
		Head:    statusData.CurrentBlock,
	}
}
