// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package og

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/types/p2p_message"
	"math/big"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/og/fetcher"
	"github.com/annchain/OG/p2p"
	log "github.com/sirupsen/logrus"

	"sync"
	"time"

	"github.com/annchain/OG/p2p/onode"
	"github.com/annchain/gcache"
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
	outgoing             chan *p2PMessage
	incoming             chan *p2PMessage
	quit                 chan bool
	CallbackRegistry     map[p2p_message.MessageType]func(*p2PMessage) // All callbacks
	CallbackRegistryOG02 map[p2p_message.MessageType]func(*p2PMessage) // All callbacks of OG02
	StatusDataProvider   NodeStatusDataProvider
	peers                *peerSet
	SubProtocols         []p2p.Protocol

	wg sync.WaitGroup // wait group is used for graceful shutdowns during downloading and processing

	messageCache gcache.Cache // cache for duplicate responses/msg to prevent storm
	maxPeers     int
	newPeerCh    chan *peer
	noMorePeers  chan struct{}
	quitSync     chan bool

	// new peer event
	OnNewPeerConnected []chan string
	Downloader         *downloader.Downloader
	Fetcher            *fetcher.Fetcher

	NodeInfo             func() *p2p.NodeInfo
	IsReceivedHash       func(hash common.Hash) bool
	broadCastMode        uint8
	encryptionPrivKey    *crypto.PrivateKey
	encryptionPubKey     *crypto.PublicKey
	disableEncryptGossip bool
}

func (h *Hub) GetBenchmarks() map[string]interface{} {
	m := map[string]interface{}{
		"outgoing":     len(h.outgoing),
		"incoming":     len(h.incoming),
		"newPeerCh":    len(h.newPeerCh),
		"messageCache": h.messageCache.Len(true),
	}
	peers := h.peers.Peers()
	for _, p := range peers {
		if p != nil {
			key := "peer_" + p.id + "_knownMsg"
			m[key] = p.knownMsg.Cardinality()
		}
	}
	return m
}

type NodeStatusDataProvider interface {
	GetCurrentNodeStatus() p2p_message.StatusData
	GetHeight() uint64
}

type PeerProvider interface {
	BestPeerInfo() (peerId string, hash common.Hash, seqId uint64, err error)
	GetPeerHead(peerId string) (hash common.Hash, seqId uint64, err error)
}

type EncryptionLayer interface {
	SetEncryptionKey(priv *crypto.PrivateKey)
}

type HubConfig struct {
	OutgoingBufferSize            int
	IncomingBufferSize            int
	MessageCacheMaxSize           int
	MessageCacheExpirationSeconds int
	MaxPeers                      int
	BroadCastMode                 uint8
	DisableEncryptGossip          bool
}

const (
	NormalMode uint8 = iota
	FeedBackMode
)

func DefaultHubConfig() HubConfig {
	config := HubConfig{
		OutgoingBufferSize:            40,
		IncomingBufferSize:            40,
		MessageCacheMaxSize:           60,
		MessageCacheExpirationSeconds: 3000,
		MaxPeers:                      50,
		BroadCastMode:                 NormalMode,
		DisableEncryptGossip:          false,
	}
	return config
}

func (h *Hub) Init(config *HubConfig) {
	h.outgoing = make(chan *p2PMessage, config.OutgoingBufferSize)
	h.incoming = make(chan *p2PMessage, config.IncomingBufferSize)
	h.peers = newPeerSet()
	h.newPeerCh = make(chan *peer)
	h.noMorePeers = make(chan struct{})
	h.quit = make(chan bool)
	h.maxPeers = config.MaxPeers
	h.quitSync = make(chan bool)
	h.messageCache = gcache.New(config.MessageCacheMaxSize).LRU().
		Expiration(time.Second * time.Duration(config.MessageCacheExpirationSeconds)).Build()
	h.CallbackRegistry = make(map[p2p_message.MessageType]func(*p2PMessage))
	h.CallbackRegistryOG02 = make(map[p2p_message.MessageType]func(*p2PMessage))
	h.broadCastMode = config.BroadCastMode
	h.disableEncryptGossip = config.DisableEncryptGossip
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
			PeerInfo: func(id onode.ID) interface{} {
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

func (h *Hub) SetEncryptionKey(priv *crypto.PrivateKey) {
	h.encryptionPrivKey = priv
	h.encryptionPubKey = priv.PublicKey()
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

	log.Debug("register peer locally")

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
	m := p2PMessage{messageType: p2p_message.MessageType(msg.Code), data: data, sourceID: p.id, version: p.version}
	//log.Debug("start handle p2p message ",p2pMsg.messageType)
	switch m.messageType {
	case p2p_message.StatusMsg:
		// Handle the message depending on its contents

		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
		// Block header query, collect the requested headers and reply
	case p2p_message.MessageTypeDuplicate:
		msgLog.WithField("got msg", m.messageType).WithField("peer ", p.String()).Trace("set path to false")
		if !p.SetOutPath(false) {
			msgLog.WithField("got msg again ", m.messageType).WithField("peer ", p.String()).Warn("set path to false")
		}
		return nil
	case p2p_message.MessageTypeSecret:
		if h.disableEncryptGossip {
			m.disableEncrypt = true
		}
		if !m.checkRequiredSize() {
			return fmt.Errorf("msg len error")
		}
		m.calculateHash()
		p.MarkMessage(m.messageType, *m.hash)
		oldMsg := m
		if duplicate := h.cacheMessage(&m); !duplicate {
			var isForMe bool
			if isForMe = m.maybeIsforMe(h.encryptionPubKey); isForMe {
				var e error
				//if encryption  gossip is disabled , just check the target
				//else decrypt
				if h.disableEncryptGossip {
					e = m.removeGossipTarget()
				} else {
					e = m.Decrypt(h.encryptionPrivKey)
				}
				if e == nil {
					err = m.Unmarshal()
					if err != nil {
						// TODO delete
						log.Errorf("unmarshal  error msg: %x", m.data)
						log.WithField("type ", m.messageType).WithError(err).Warn("handle msg error")
						return err
					}
					msgLog.WithField("type", m.messageType.String()).WithField("from", p.String()).WithField(
						"Message", m.message.String()).WithField("len ", len(m.data)).Debug("received a message")
					h.incoming <- &m
				} else {
					log.WithError(e).Debug("this msg is not for me, will relay")
					isForMe = false
				}
			}
			log.Debug("this msg is not for me, will relay")
			h.RelayMessage(&oldMsg)

		} else {
			out, in := p.CheckPath()
			log.WithField("type", m.messageType).WithField("size", len(m.data)).WithField(
				"hash", m.hash).WithField("from ", p.String()).WithField("out ", out).WithField(
				"in ", in).Debug("duplicate msg ,discard")

		}
		return nil
	default:
		//for incoming msg
		err = m.Unmarshal()
		if err != nil {
			log.WithField("type ", m.messageType).WithError(err).Warn("handle msg error")
			return err
		}
		m.calculateHash()
		p.MarkMessage(m.messageType, *m.hash)
		hashes := m.GetMarkHashes()
		if len(hashes) != 0 {
			msgLog.WithField("len hahses", len(hashes)).Trace("before mark msg")
			for _, hash := range hashes {
				p.MarkMessage(p2p_message.MessageTypeNewTx, hash)
			}
			msgLog.WithField("len ", len(hashes)).Trace("after mark msg")
		}
		if duplicate := h.cacheMessage(&m); duplicate {
			out, in := p.CheckPath()
			log.WithField("type", m.messageType).WithField("msg", m.message.String()).WithField(
				"hash", m.hash).WithField("from ", p.String()).WithField("out ", out).WithField(
				"in ", in).Debug("duplicate msg ,discard")
			if h.broadCastMode == FeedBackMode && m.sendDuplicateMsg() {
				if outNum, inNum := h.peers.ValidPathNum(); inNum <= 1 {
					log.WithField("outNum ", outNum).WithField("inNum", inNum).Debug("not enough valid path")
					//return nil
				}
				var dup p2p_message.MessageDuplicate
				p.SetInPath(false)
				return h.SendToPeer(p.id, p2p_message.MessageTypeDuplicate, &dup)
			}
			return nil
		}
	}

	msgLog.WithField("type", m.messageType.String()).WithField("from", p.String()).WithField(
		"Message", m.message.String()).WithField("len ", len(m.data)).Debug("received a message")

	h.incoming <- &m
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
	goroutine.New(h.loopSend)
	goroutine.New(h.loopReceive)
	goroutine.New(h.loopNotify)
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
			//bug fix , don't use go routine to send here
			// start a new routine in order not to block other communications
			switch m.sendingType {
			case sendingTypeBroacast:
				h.broadcastMessage(m)
			case sendingTypeMulticast:
				h.multicastMessage(m)
			case sendingTypeMulticastToSource:
				h.multicastMessageToSource(m)
			case sendingTypeBroacastWithFilter:
				h.broadcastMessage(m)
			case sendingTypeBroacastWithLink:
				h.broadcastMessageWithLink(m)

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
			// bug fix ; don't use go routine
			h.receiveMessage(m)
		case <-h.quit:
			log.Info("Hub-loopReceive received quit message. Quitting...")
			return
		}
	}
}

//MulticastToSource  multicast msg to source , for example , send tx request to the peer which hash the tx
func (h *Hub) MulticastToSource(messageType p2p_message.MessageType, msg p2p_message.Message, sourceMsgHash *common.Hash) {
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeMulticastToSource, sourceHash: sourceMsgHash}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("multicast msg to source")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessage(messageType p2p_message.MessageType, msg p2p_message.Message) {
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeBroacast}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessageWithLink(messageType p2p_message.MessageType, msg p2p_message.Message) {
	if h.broadCastMode != FeedBackMode {
		msgLog.WithField("type", messageType).Trace("broadcast withlink disabled")
		h.BroadcastMessage(messageType, msg)
		return
	}
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeBroacastWithLink}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//BroadcastMessage broadcast to whole network
func (h *Hub) BroadcastMessageWithFilter(messageType p2p_message.MessageType, msg p2p_message.Message) {
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeBroacastWithFilter}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("size ", len(msgOut.data)).WithField("type", messageType).Debug("broadcast message")
	h.outgoing <- msgOut
}

//MulticastMessage multicast message to some peer
func (h *Hub) MulticastMessage(messageType p2p_message.MessageType, msg p2p_message.Message) {
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeMulticast}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("broadcast message init msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("size", len(msgOut.data)).WithField("type", messageType).Trace("multicast message")
	h.outgoing <- msgOut
}

//SendToAnynomous send msg by  Anynomous
func (h *Hub) SendToAnynomous(messageType p2p_message.MessageType, msg p2p_message.Message, anyNomousPubKey *crypto.PublicKey) {
	msgOut := &p2PMessage{messageType: messageType, message: msg, sendingType: sendingTypeBroacast}
	if h.disableEncryptGossip {
		msgOut.disableEncrypt = true
	}
	err := msgOut.Marshal()
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("SendToAnynomous message init msg  err")
		return
	}
	beforeEncSize := len(msgOut.data)
	if h.disableEncryptGossip {
		err = msgOut.appendGossipTarget(anyNomousPubKey)
	} else {
		err = msgOut.Encrypt(anyNomousPubKey)
	}
	if err != nil {
		msgLog.WithError(err).WithField("type", messageType).Warn("SendToAnynomous message encrypt msg  err")
		return
	}
	msgOut.calculateHash()
	msgLog.WithField("before enc size ", beforeEncSize).WithField("size", len(msgOut.data)).WithField("type", messageType).Trace("SendToAnynomous message")
	h.outgoing <- msgOut

}

func (h *Hub) RelayMessage(msgOut *p2PMessage) {
	msgLog.WithField("size", len(msgOut.data)).WithField("type", msgOut.messageType).Trace("relay message")
	h.outgoing <- msgOut
}

func (h *Hub) SendToPeer(peerId string, messageType p2p_message.MessageType, msg p2p_message.Message) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.sendRequest(messageType, msg)
}

func (h *Hub) SendGetMsg(peerId string, msg *p2p_message.MessageGetMsg) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		ids := h.getMsgFromCache(p2p_message.MessageTypeControl, *msg.Hash)
		ps := h.peers.GetPeers(ids, 1)
		if len(ps) != 0 {
			p = ps[0]
		}
	}
	if p == nil {
		h.MulticastMessage(p2p_message.MessageTypeGetMsg, msg)
		return nil
	} else {
		p.SetInPath(true)
	}
	return p.sendRequest(p2p_message.MessageTypeGetMsg, msg)
}

func (h *Hub) SendBytesToPeer(peerId string, messageType p2p_message.MessageType, msg []byte) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.sendRawMessage(messageType, msg)
}

// SetPeerHead is just a hack to set the latest seq number known of the peer
// This value ought not to be stored in peer, but an outside map.
// This has nothing related to p2p.
func (h *Hub) SetPeerHead(peerId string, hash common.Hash, number uint64) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	p.SetHead(hash, number)
	return nil
}

func (h *Hub) BestPeerInfo() (peerId string, hash common.Hash, seqId uint64, err error) {
	p := h.peers.BestPeer()
	if p != nil {
		peerId = p.id
		hash, seqId = p.Head()
		return
	}
	err = fmt.Errorf("no best peer")
	return
}

func (h *Hub) GetPeerHead(peerId string) (hash common.Hash, seqId uint64, err error) {
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

//broadcastMessage
func (h *Hub) broadcastMessage(msg *p2PMessage) {
	var peers []*peer
	// choose all  peer and then send.
	peers = h.peers.PeersWithoutMsg(*msg.hash, msg.messageType)
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
	return
}

func (h *Hub) broadcastMessageWithLink(msg *p2PMessage) {
	var peers []*peer
	// choose all  peer and then send.
	var hash common.Hash
	hash = *msg.hash
	c := p2p_message.MessageControl{Hash: &hash}
	var pMsg = &p2PMessage{messageType: p2p_message.MessageTypeControl, message: &c}
	//outgoing msg
	if err := pMsg.Marshal(); err != nil {
		msgLog.Error(err)
	}
	pMsg.calculateHash()
	peers = h.peers.PeersWithoutMsg(*msg.hash, msg.messageType)
	for _, peer := range peers {
		//if  len(peers) == 1 {
		if out, _ := peer.CheckPath(); out {
			msgLog.WithField("peer ", peer.String()).Debug("send original tx to peer")
			peer.AsyncSendMessage(msg)
		} else {
			msgLog.WithField("to peer ", peer.String()).WithField("hash ", hash).Debug("send MessageTypeControl")
			peer.AsyncSendMessage(pMsg)
		}
	}
	return
}

/*
func (h *Hub) broadcastMessageWithFilter(msg *p2PMessage) {
	newSeq := msg.Message.(*p2p_message.MessageNewSequencer)
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
func (h *Hub) multicastMessage(msg *p2PMessage) error {
	peers := h.peers.GetRandomPeers(3)
	// choose random peer and then send.
	for _, peer := range peers {
		peer.AsyncSendMessage(msg)
	}
	return nil
}

//multicastMessageToSource
func (h *Hub) multicastMessageToSource(msg *p2PMessage) error {
	if msg.sourceHash == nil {
		msgLog.Warn("source msg hash is nil , multicast to random ")
		return h.multicastMessage(msg)
	}
	ids := h.getMsgFromCache(p2p_message.MessageTypeControl, *msg.sourceHash)
	if len(ids) == 0 {
		ids = h.getMsgFromCache(p2p_message.MessageTypeNewTx, *msg.sourceHash)
	}
	//send to 2 peer , considering if one peer disconnect,
	peers := h.peers.GetPeers(ids, 3)
	if len(peers) == 0 {
		msgLog.WithField("type ", msg.messageType).WithField("peeers id ", ids).Warn(
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
func (h *Hub) cacheMessage(m *p2PMessage) (exists bool) {
	if m.hash == nil {
		return false
	}
	var peers []string
	key := m.msgKey()
	if a, err := h.messageCache.GetIFPresent(key); err == nil {
		// already there
		exists = true
		//var peers []string
		peers = a.([]string)
		msgLog.WithField("from ", m.sourceID).WithField("hash", m.hash).WithField("peers", peers).WithField(
			"type", m.messageType).Trace("we have a duplicate message. Discard")
		if len(peers) == 0 {
			msgLog.Error("peers is nil")
		} else if len(peers) >= DuplicateMsgPeerNum {
			return exists
		}
	}
	peers = append(peers, m.sourceID)
	h.messageCache.Set(key, peers)
	return exists
}

//getMsgFromCache
func (h *Hub) getMsgFromCache(m p2p_message.MessageType, hash common.Hash) []string {
	key := p2p_message.NewMsgKey(m, hash)
	if a, err := h.messageCache.GetIFPresent(key); err == nil {
		var peers []string
		peers = a.([]string)
		msgLog.WithField("peers ", peers).Trace("get peers from cache ")
		return peers
	}
	return nil
}

func (h *Hub) receiveMessage(msg *p2PMessage) {
	// route to specific callbacks according to the registry.
	if msg.version >= OG02 {
		if v, ok := h.CallbackRegistryOG02[msg.messageType]; ok {
			log.WithField("from", msg.sourceID).WithField("type", msg.messageType.String()).Trace("Received a message")
			v(msg)
			return
		}
	}
	if msg.messageType == p2p_message.MessageTypeGetMsg {
		peer := h.peers.Peer(msg.sourceID)
		if peer != nil {
			msgLog.WithField("msg", msg.message.String()).WithField("peer ", peer.String()).Trace("set path to true")
			peer.SetOutPath(true)
		}
	}

	if v, ok := h.CallbackRegistry[msg.messageType]; ok {
		msgLog.WithField("type", msg.messageType).Debug("handle")
		v(msg)
	} else {
		msgLog.WithField("from", msg.sourceID).WithField("type", msg.messageType).Debug("Received an Unknown message")
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

// NodeInfo represents a short summary of the OG sub-protocol metadata
// known about the host peer.
type NodeStatus struct {
	Network    uint64      `json:"network"`    // OG network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int    `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	Head       common.Hash `json:"head"`       // SHA3 hash of the host's best owned block
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
