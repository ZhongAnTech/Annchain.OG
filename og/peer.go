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
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/msg"
	"sync"
	"time"

	"github.com/annchain/OG/p2p"
	"github.com/deckarep/golang-set"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxknownMsg = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)

	// maxqueuedMsg is the maximum number of transaction lists to queue up before
	// dropping broadcasts. This is a sensitive number as a transaction list might
	// contain a single transaction, or thousands.
	maxqueuedMsg = 128

	// maxQueuedProps is the maximum number of block propagations to queue up before
	// dropping broadcasts. There's not much point in queueing stale blocks, so a few
	// that might cover uncles should be enough.
	maxQueuedProps = 4

	// maxQueuedAnns is the maximum number of block announcements to queue up before
	// dropping broadcasts. Similarly to block propagations, there's no point to queue
	// above some healthy uncle limit, so use that.
	maxQueuedAnns = 4

	handshakeTimeout = 5 * time.Second
)

type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         // Protocol Version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	head      common.Hash
	seqId     uint64
	lock      sync.RWMutex
	knownMsg  mapset.Set           // Set of transaction hashes known to be known by this peer
	queuedMsg chan []msg.OgMessage // Queue of transactions to broadcast to the peer
	term      chan struct{}        // Termination channel to stop the broadcaster
	outPath   bool
	inPath    bool
	inBound   bool
}

type PeerInfo struct {
	Version     int    `json:"Version"`      // Ethereum protocol Version negotiated
	SequencerId uint64 `json:"sequencer_id"` // Total difficulty of the peer's blockchain
	Head        string `json:"head"`         // SHA3 Hash of the peer's best owned block
	ShortId     string `json:"short_id"`
	Link        bool   `json:"link"`
	Addrs       string `json:"addrs"`
	InBound     bool   `json:"in_bound"`
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:      p,
		rw:        rw,
		version:   version,
		id:        fmt.Sprintf("%x", p.ID().Bytes()[:8]),
		knownMsg:  mapset.NewSet(),
		queuedMsg: make(chan []msg.OgMessage, maxqueuedMsg),
		term:      make(chan struct{}),
		outPath:   true,
		inPath:    true,
		inBound:   p.Inbound(),
	}
}

// broadcast is a write loop that multiplexes block propagations, announcements
// and transaction broadcasts into the remote peer. The goal is to have an async
// writer that does not lock up node internals.
func (p *peer) broadcast() {
	for {
		select {
		case msg := <-p.queuedMsg:
			if err := p.SendMessages(msg); err != nil {
				msgLog.WithError(err).Warn("send msg failed,quiting")
				return
			}
			msgLog.WithField("count", len(msg)).Trace("Broadcast messages")

		case <-p.term:
			msgLog.Debug("peer terminating,quiting")
			return
		}
	}
}

func (p *peer) SetOutPath(v bool) bool {
	if p.outPath == v {
		return false
	}
	p.outPath = v
	return true
}

func (p *peer) SetInPath(v bool) {
	p.inPath = v
}

func (p *peer) CheckPath() (outPath, inpath bool) {
	return p.outPath, p.inPath
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, seqId := p.Head()

	return &PeerInfo{
		Version:     p.version,
		Head:        hash.Hex(),
		SequencerId: seqId,
		ShortId:     p.id,
		Link:        p.outPath,
		Addrs:       p.RemoteAddr().String(),
		InBound:     p.inBound,
	}
}

// Head retrieves a copy of the current head Hash and total difficulty of the
// peer.
func (p *peer) Head() (hash common.Hash, seqId uint64) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash.Bytes[:], p.head.Bytes[:])
	return hash, p.seqId
}

// SetHead updates the head Hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, seqId uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head.Bytes[:], hash.Bytes[:])
	p.seqId = seqId
}

// MarkMessage marks a Message as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkMessage(m message.BinaryMessageType, hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction Hash
	for p.knownMsg.Cardinality() >= maxknownMsg {
		p.knownMsg.Pop()
	}
	key := message.NewMsgKey(m, hash)
	p.knownMsg.Add(key)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction Hash set for future reference.
func (p *peer) SendMessages(messages []msg.OgMessage) error {
	var msgType msg.BinaryMessageType
	var msgBytes []byte
	if len(messages) == 0 {
		return nil
	}
	for _, msg := range messages {
		//duplicated
		//key := msg.MsgKey()
		//p.knownMsg.Add(key)
		msgType = msg.MessageType
		msgBytes = append(msgBytes, msg.Data...)
	}
	return p.sendRawMessage(msgType, msgBytes)
}

func (p *peer) sendRawMessage(msgType msg.BinaryMessageType, msgBytes []byte) error {
	msgLog.WithField("to ", p.id).WithField("type ", msgType).WithField("size", len(msgBytes)).Trace("send msg")
	return p2p.Send(p.rw, msgType.Code(), msgBytes)

}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendMessages(messages []*message.types) {
	select {
	case p.queuedMsg <- messages:
		for _, msg := range messages {
			key := msg.MsgKey()
			p.knownMsg.Add(key)
		}
	default:
		msgLog.WithField("count", len(messages)).Debug("Dropping transaction propagation")
	}
}

func (p *peer) AsyncSendMessage(msg *message.types) {
	var messages []*message.types
	messages = append(messages, msg)
	select {
	case p.queuedMsg <- messages:
		key := msg.MsgKey()
		p.knownMsg.Add(key)
	default:
		msgLog.Debug("Dropping transaction propagation")
	}
}

// SendNodeData sends a batch of arbitrary internal Data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data []byte) error {
	return p.sendRawMessage(message.NodeDataMsg, data)
}

// RequestNodeData fetches a batch of arbitrary Data from a node's known state
// Data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes common.Hashes) error {
	msgLog.WithField("count", len(hashes)).Debug("Fetching batch of state Data")
	hashsStruct := common.Hashes(hashes)
	b, _ := hashsStruct.MarshalMsg(nil)
	return p.sendRawMessage(message.GetNodeDataMsg, b)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes common.Hashes) error {
	msgLog.WithField("count", len(hashes)).Debug("Fetching batch of receipts")
	b, _ := hashes.MarshalMsg(nil)
	return p.sendRawMessage(message.GetReceiptsMsg, b)
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the Hash of an origin block.
func (p *peer) RequestTxsByHash(seqHash common.Hash, seqId uint64) error {
	hash := seqHash
	msg := &p2p_message.MessageTxsRequest{
		SeqHash:   &hash,
		Id:        &seqId,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeTxsRequest, msg)
}

func (p *peer) RequestTxs(hashs common.Hashes) error {
	msg := &p2p_message.MessageTxsRequest{
		Hashes:    &hashs,
		RequestId: message.MsgCounter.Get(),
	}

	return p.sendRequest(message.MessageTypeTxsRequest, msg)
}

func (p *peer) RequestTxsById(seqId uint64) error {
	msg := &p2p_message.MessageTxsRequest{
		Id:        &seqId,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeTxsRequest, msg)
}

func (p *peer) RequestBodies(seqHashs common.Hashes) error {
	msg := &p2p_message.MessageBodiesRequest{
		SeqHashes: seqHashs,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeBodiesRequest, msg)
}

func (h *Hub) RequestOneHeader(peerId string, hash common.Hash) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.RequestOneHeader(hash)
}

func (h *Hub) RequestBodies(peerId string, hashs common.Hashes) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.RequestBodies(hashs)
}

func (p *peer) RequestOneHeader(hash common.Hash) error {
	tmpHash := hash
	msg := &p2p_message.MessageHeaderRequest{
		Origin: p2p_message.HashOrNumber{
			Hash: &tmpHash,
		},
		Amount:    uint64(1),
		Skip:      uint64(0),
		Reverse:   false,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeHeaderRequest, msg)
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	msg := &p2p_message.MessageHeaderRequest{
		Origin: p2p_message.HashOrNumber{
			Number: &origin,
		},
		Amount:    uint64(amount),
		Skip:      uint64(skip),
		Reverse:   reverse,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeHeaderRequest, msg)
}

func (p *peer) RequestHeadersByHash(hash common.Hash, amount int, skip int, reverse bool) error {
	tmpHash := hash
	msg := &p2p_message.MessageHeaderRequest{
		Origin: p2p_message.HashOrNumber{
			Hash: &tmpHash,
		},
		Amount:    uint64(amount),
		Skip:      uint64(skip),
		Reverse:   reverse,
		RequestId: message.MsgCounter.Get(),
	}
	return p.sendRequest(message.MessageTypeHeaderRequest, msg)
}

func (p *peer) sendRequest(msgType message.BinaryMessageType, request p2p_message.Message) error {
	clog := msgLog.WithField("msgType", msgType).WithField("request ", request).WithField("to", p.id)
	data, err := request.MarshalMsg(nil)
	if err != nil {
		clog.WithError(err).Warn("encode request error")
		return err
	}
	clog.WithField("size", len(data)).Debug("send")
	err = p2p.Send(p.rw, p2p.MsgCodeType(msgType), data)
	if err != nil {
		clog.WithError(err).Warn("send failed")
	}
	return err
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer-%s-[%s]-[%s]", p.id,
		fmt.Sprintf("og/%2d", p.version), p.RemoteAddr().String(),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	goroutine.New(p.broadcast)

	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	p, ok := ps.peers[id]
	if !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	p.close()

	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

func (ps *peerSet) Peers() []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}

func (ps *peerSet) ValidPathNum() (outPath, inPath int) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	var out, in int
	for _, p := range ps.peers {
		if p.outPath {
			out++
		}
		if p.inPath {
			in++
		}
	}
	return out, in
}

func (ps *peerSet) GetPeers(ids []string, n int) []*peer {
	if len(ids) == 0 || n <= 0 {
		return nil
	}
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	all := make([]*peer, 0, len(ids))
	list := make([]*peer, 0, n)
	for _, id := range ids {
		peer := ps.peers[id]
		if peer != nil {
			all = append(all, peer)
		}
	}
	indices := math.GenerateRandomIndices(n, len(all))
	for _, i := range indices {
		list = append(list, all[i])
	}
	return list
}

func (ps *peerSet) GetRandomPeers(n int) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	all := make([]*peer, 0, len(ps.peers))
	list := make([]*peer, 0, n)
	for _, p := range ps.peers {
		all = append(all, p)
	}
	indices := math.GenerateRandomIndices(n, len(all))
	for _, i := range indices {
		list = append(list, all[i])
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutMsg(hash common.Hash, m message.BinaryMessageType) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	if m == message.MessageTypeNewTx || m == message.MessageTypeNewSequencer {
		keyControl := message.NewMsgKey(message.MessageTypeControl, hash)
		key := message.NewMsgKey(m, hash)
		for _, p := range ps.peers {
			if !p.knownMsg.Contains(key) && !p.knownMsg.Contains(keyControl) {
				list = append(list, p)
			}
		}
		return list
	}
	key := message.NewMsgKey(m, hash)
	for _, p := range ps.peers {
		if !p.knownMsg.Contains(key) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestId   uint64
	)
	for _, p := range ps.peers {
		if _, seqId := p.Head(); bestPeer == nil || seqId > bestId {
			bestPeer, bestId = p, seqId
		}
	}
	return bestPeer
}

// Close disconnects all peers.
// No new peers can be registered after Close has returned.
func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}

// Handshake executes the og protocol handshake, negotiating Version number,
// network IDs, head and genesis blocks.
func (p *peer) Handshake(network uint64, head common.Hash, seqId uint64, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status message.StatusData // safe to read after two values have been received from errc

	sendStatusFunc := func() {
		s := message.StatusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			CurrentBlock:    head,
			CurrentId:       seqId,
			GenesisBlock:    genesis,
		}
		data, _ := s.MarshalMsg(nil)
		errc <- p2p.Send(p.rw, p2p.MsgCodeType(message.StatusMsg), data)
	}
	goroutine.New(sendStatusFunc)
	readFunc := func() {
		errc <- p.readStatus(network, &status, genesis)
	}
	goroutine.New(readFunc)
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	p.head = status.CurrentBlock
	p.seqId = status.CurrentId
	return nil
}

func (p *peer) readStatus(network uint64, status *message.StatusData, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != p2p.MsgCodeType(message.StatusMsg) {
		return ErrResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, message.StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return ErrResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	data, err := msg.GetPayLoad()
	if err != nil {
		return ErrResp(ErrDecode, "msg %v: %v", msg, err)
	}
	_, err = status.UnmarshalMsg(data)
	// Decode the handshake and make sure everything matches
	if err != nil {
		return ErrResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return ErrResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock.Bytes[:8], genesis.Bytes[:8])
	}
	if status.NetworkId != network {
		return ErrResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return ErrResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}
