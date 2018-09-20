package og

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/og/fetcher"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"math/big"


	"github.com/bluele/gcache"
	"sync"
	"sync/atomic"
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
// Hub will also prevent duplicate requests/responses.
// If there is any failure, Hub is NOT responsible for changing a peer and retry. (maybe enhanced in the future.)
type Hub struct {
	outgoing         chan *P2PMessage
	incoming         chan *P2PMessage
	quit             chan bool
	CallbackRegistry map[MessageType]func(*P2PMessage) // All callbacks
	peers            *peerSet
	SubProtocols     []p2p.Protocol


	wg               sync.WaitGroup // wait group is used for graceful shutdowns during downloading and processing

	Dag              *core.Dag
	messageCache     gcache.Cache // cache for duplicate responses/msg to prevent storm


	maxPeers    int
	fastSync    uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs   uint32 // Flag whether we're considered synchronised (enables transaction processing)
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh chan *peer
	txsyncCh  chan *txsync

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher


	networkID  uint64
	TxBuffer   *TxBuffer
	SyncBuffer *SyncBuffer

}

type HubConfig struct {
	OutgoingBufferSize            int
	IncomingBufferSize            int
	MessageCacheMaxSize           int
	MessageCacheExpirationSeconds int
}

func (h*Hub)Set(dag *core.Dag) {
	h.downloader.Cancel()
}


func (h *Hub) Init(config *HubConfig, maxPeer int,networkId uint64,dag *core.Dag) {

	h.outgoing = make(chan *P2PMessage, config.OutgoingBufferSize)
	h.incoming = make(chan *P2PMessage, config.IncomingBufferSize)
	h.quit = make(chan bool)
	h.peers = newPeerSet()
	h.newPeerCh = make(chan *peer)
	h.noMorePeers = make(chan struct{})
	h.txsyncCh=    make(chan *txsync)
	h.quitSync=    make(chan struct{})
	h.maxPeers = maxPeer
    h.networkID = networkId
    h.Dag = dag
	h.messageCache = gcache.New(config.MessageCacheMaxSize).LRU().
		Expiration(time.Second * time.Duration(config.MessageCacheExpirationSeconds)).Build()
	h.CallbackRegistry = make(map[MessageType]func(*P2PMessage))
}

func NewHub(config *HubConfig, maxPeer int, mode downloader.SyncMode, networkID uint64, dag *core.Dag) *Hub {
	h := &Hub{}
	h.Init(config, maxPeer, networkID,dag )
	// Figure out whether to allow fast sync or not
	if mode == downloader.FastSync && h.Dag.LatestSequencer().Id > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		h.fastSync = uint32(1)
	}
	h.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if mode == downloader.FastSync && version < OG32 {
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
				case <-h.quitSync:
					return p2p.DiscQuitting
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
	// Construct the different synchronisation mechanisms

	h.downloader = downloader.New(mode, h.Dag, h.removePeer)
	heighter := func() uint64 {
		return h.Dag.LatestSequencer().Id
	}
	inserter := func(tx types.Txi) error {
		// If fast sync is running, deny importing weird blocks
		if atomic.LoadUint32(&h.fastSync) == 1 {
			log.WithField("number", tx.GetHeight()).WithField( "hash", tx.GetTxHash()).Warn("Discarded bad propagated sequencer")
			return nil
		}
		atomic.StoreUint32(&h.acceptTxs, 1) // Mark initial sync done on any fetcher import
		h.TxBuffer.AddTx(tx)
		return nil
	}
	h.fetcher = fetcher.New(h.GetSequencerByHash, heighter, inserter, h.removePeer)

	return h
}

func (h *Hub) AddTxs(txs types.Txs, seq *types.Sequencer) error {
	var txis []types.Txi
	for _, tx := range txs {
		txis = append(txis, tx)
	}
	txis = append(txis, seq)
	go h.SyncBuffer.AddTxs(txis)
	return nil
}

func (h *Hub) GetSequencerByHash(hash types.Hash) *types.Sequencer {
	txi := h.Dag.GetTx(hash)
	switch tx := txi.(type) {
	case *types.Sequencer:
		return tx
		break
	default:
		return nil
	}
	return nil
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
	var (
		genesis = h.Dag.Genesis()
		lastSeq = h.Dag.LatestSequencer()
		head    = types.Hash{}
	)
	if lastSeq == nil {
		panic("Last sequencer is nil")
	} else {
		head = lastSeq.Hash
	}
	if err := p.Handshake(h.networkID, head, genesis.Hash); err != nil {
		log.WithError(err).WithField("peer ",p.id).Debug("OG handshake failed")
		return err
	}
	// Register the peer locally
	if err := h.peers.Register(p); err != nil {
		log.WithError(err).Error("og peer registration failed")
		return err
	}
	log.Debug("register peer localy")
	defer h.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := h.downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	h.syncTransactions(p)

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
	p2pMsg := P2PMessage{MessageType: MessageType(msg.Code), Message: data}
	log.Debug("start handle p2p messgae ",p2pMsg.MessageType)
	switch {
	case p2pMsg.MessageType == StatusMsg:
		// Handle the message depending on its contentsms

		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
		// Block header query, collect the requested headers and reply
	case p2pMsg.MessageType == MessageTypeHeaderRequest:
		log.Debug("got MessageTypeHeaderRequest")
		// Decode the complex header query
		var query types.MessageHeaderRequest
		if _, err := query.UnmarshalMsg(p2pMsg.Message); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := !query.Origin.Hash.Empty()
		first := true
        log.WithField("hash",query.Origin.Hash).WithField("number",query.Origin.Number).WithField(
        	"hashmode",hashMode).WithField("amount",query.Amount).WithField("skip",query.Skip).Debug("requests")
		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Sequencer
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Sequencer
			if hashMode {
				if first {
					first = false
					origin = h.Dag.GetSequencerByHash(query.Origin.Hash)
					if origin != nil {
						query.Origin.Number = origin.Number()
					}
				} else {
					origin = h.Dag.GetSequencer(query.Origin.Hash, query.Origin.Number)
				}
			} else {
				origin = h.Dag.GetSequencerById(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case hashMode && query.Reverse:
				// Hash based traversal towards the genesis block
				ancestor := query.Skip + 1
				if ancestor == 0 {
					unknown = true
				} else {
					seq := h.Dag.GetSequencerById(query.Origin.Number - ancestor)
					query.Origin.Hash, query.Origin.Number = seq.GetTxHash(), seq.Number()
					unknown = (query.Origin.Hash.Empty())
				}
			case hashMode && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					log.Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := h.Dag.GetSequencerById(next); header != nil {
						nextHash := header.GetTxHash()
						oldSeq := h.Dag.GetSequencerById(next - (query.Skip + 1))
						expOldHash := oldSeq.GetTxHash()
						if expOldHash == query.Origin.Hash {
							query.Origin.Hash, query.Origin.Number = nextHash, next
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= query.Skip + 1
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += query.Skip + 1
			}
		}

		msgRes := types.MessageHeaderResponse{
			Sequencers: headers,
		}
		data, _ := msgRes.MarshalMsg(nil)
		log.Debug("send MessageTypeGetHeader")
		return p.sendRawMessage(uint64(MessageTypeGetHeader), data)
	case p2pMsg.MessageType == MessageTypeGetHeader:
		log.Debug("got MessageTypeGetHeader")
		// A batch of headers arrived to one of our previous requests
		var headerMsg types.MessageHeaderResponse
		if _, err := headerMsg.UnmarshalMsg(p2pMsg.Message); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		headers := headerMsg.Sequencers
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		seqHeaders := types.SeqsToHeaders(headers)
		filter := len(seqHeaders) == 1

		if filter {
			// Irrelevant of the fork checks, send the header to the fetcher just in case

			seqHeaders = h.fetcher.FilterHeaders(p.id, seqHeaders, time.Now())
		}
		if len(seqHeaders) > 0 || !filter {
			err := h.downloader.DeliverHeaders(p.id, seqHeaders)
			if err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}
		log.Debug("heandle  MessageTypeGetHeader")

	case p2pMsg.MessageType == MessageTypeTxsRequest:
		// Decode the retrieval message
		log.Debug("got MessageTypeTxsRequest")
		var msgReq types.MessageTxsRequest
		if _, err := msgReq.UnmarshalMsg(p2pMsg.Message); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		var msgRes types.MessageNewSyncTxsResponse

		var seq *types.Sequencer
		if msgReq.SeqHash != nil && msgReq.Id != 0 {
			seq = h.Dag.GetSequencer(*msgReq.SeqHash, msgReq.Id)
		} else {
			seq = h.Dag.GetSequencerById(msgReq.Id)
		}
		if seq == nil {
			return nil
		}
		msgRes.Sequencer = seq
		msgRes.Txs = h.Dag.GetTxsByNumber(msgReq.Id)
		data, _ := msgRes.MarshalMsg(nil)
		log.Debug("send MessageTypeGetTxs")
		return p.sendRawMessage(uint64(MessageTypeGetTxs), data)

	case p2pMsg.MessageType == MessageTypeGetTxs:
		log.Debug("got MessageTypeGetTxs")
		// A batch of block bodies arrived to one of our previous requests
		var request types.MessageNewSyncTxsResponse
		if _, err := request.UnmarshalMsg(p2pMsg.Message); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver them all to the downloader for queuing
		transactions := make([][]*types.Tx, 1)
		transactions[0] = request.Txs

		// Filter out any explicitly requested bodies, deliver the rest to the downloader
		filter := len(transactions) > 0
		if filter {
			transactions = h.fetcher.FilterBodies(p.id, transactions, request.Sequencer, time.Now())
		}
		if len(transactions) > 0 || !filter {
			err := h.downloader.DeliverBodies(p.id, transactions, nil)
			if err != nil {
				log.Debug("Failed to deliver bodies", "err", err)
			}
		}
		log.Debug("handle MessageTypeGetTxs")
		return nil

	case p2pMsg.MessageType == MessageTypeNewSequence:

		var msgseq types.MessageNewSequence
		_, e := msgseq.UnmarshalMsg(p2pMsg.Message)
		if e == nil && msgseq.Sequencer != nil {
			//set peer's head
			p.SetHead(msgseq.Sequencer.GetTxHash(), msgseq.Sequencer.Id)
		}
		//dont accept new seq until we catch up
		if atomic.LoadUint32(&h.acceptTxs) == 0 {
			return nil
		}
	case p2pMsg.MessageType == MessageTypeNewTx && atomic.LoadUint32(&h.acceptTxs) == 0:
		// no receive until sync finish
		return nil
	case p.version >= OG32 && p2pMsg.MessageType == GetNodeDataMsg:
		log.Warn("got GetNodeDataMsg ")
		//todo
		//p.SendNodeData(nil)
		log.Debug("need send node data")


	case p.version >= OG32 && p2pMsg.MessageType == NodeDataMsg:
		// Deliver all to the downloader
		if err := h.downloader.DeliverNodeData(p.id, nil); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}

	default:
		log.Debug("got default message type ",p2pMsg.MessageType)
		p2pMsg.init()
		if p2pMsg.needCheckRepeat {
			p.MarkMessage(p2pMsg.hash)
		}
		h.incoming <- &p2pMsg
		return nil
	}

	return nil
}

func (h *Hub) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := h.peers.Peer(id)
	if peer == nil {
		log.Debug("peer not found id")
		return
	}
	log.WithField("peer", id).Debug("Removing og peer")

	// Unregister the peer from the downloader and OG peer set
	h.downloader.UnregisterPeer(id)
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
	//h.downloader.Start()
	go h.loopSend()
	go h.loopReceive()

	// start sync handlers
	go h.syncer()
	go h.txsyncLoop()
}

func (h *Hub) Stop() {

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	h.noMorePeers <- struct{}{}
	// Quit fetcher, txsyncLoop.
	close(h.quitSync)
	h.quit <- true
	h.peers.Close()
	h.wg.Wait()

	log.Info("hub stopped")
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
			// check duplicates
			if _, err := h.messageCache.GetIFPresent(m.hash); err == nil {
				// already there
				log.WithField("hash", m.hash).WithField("type", m.MessageType.String()).
					Warn("we have a duplicate message. Discard")
				continue
			}
			h.messageCache.Set(m.hash, nil)
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
	msgOut := &P2PMessage{MessageType: messageType, Message: msg}
	log.WithField("type", messageType).Info("sending message")
	h.outgoing <- msgOut
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
		log.WithField("type", msg.MessageType.String()).Debug("Received a message")
		v(msg)
	} else {
		log.WithField("type", msg.MessageType).Debug("Received an Unknown message")
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
	return &NodeInfo{
		Network: h.networkID,
		Genesis: types.Hash{},
		Head:    types.Hash{},
	}
}
