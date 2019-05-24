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
	"github.com/annchain/OG/common/goroutine"
	"math/rand"
	"sync"
	"time"

	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
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

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	head      types.Hash
	seqId     uint64
	lock      sync.RWMutex
	knownMsg  mapset.Set         // Set of transaction hashes known to be known by this peer
	queuedMsg chan []*p2PMessage // Queue of transactions to broadcast to the peer
	term      chan struct{}      // Termination channel to stop the broadcaster
	outPath   bool
	inPath    bool
	inBound   bool
}

type PeerInfo struct {
	Version     int    `json:"version"`      // Ethereum protocol version negotiated
	SequencerId uint64 `json:"sequencer_id"` // Total difficulty of the peer's blockchain
	Head        string `json:"head"`         // SHA3 hash of the peer's best owned block
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
		queuedMsg: make(chan []*p2PMessage, maxqueuedMsg),
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

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash types.Hash, seqId uint64) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash.Bytes[:], p.head.Bytes[:])
	return hash, p.seqId
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash types.Hash, seqId uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head.Bytes[:], hash.Bytes[:])
	p.seqId = seqId
}

// MarkMessage marks a message as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkMessage(m MessageType, hash types.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownMsg.Cardinality() >= maxknownMsg {
		p.knownMsg.Pop()
	}
	key := newMsgKey(m, hash)
	p.knownMsg.Add(key)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendMessages(messages []*p2PMessage) error {
	var msgType MessageType
	var msgBytes []byte
	if len(messages) == 0 {
		return nil
	}
	for _, msg := range messages {
		key := msg.msgKey()
		p.knownMsg.Add(key)
		msgType = msg.messageType
		msgBytes = append(msgBytes, msg.data...)
	}
	return p.sendRawMessage(msgType, msgBytes)
}

func (p *peer) sendRawMessage(msgType MessageType, msgBytes []byte) error {
	msgLog.WithField("to ", p.id).WithField("type ", msgType).WithField("size", len(msgBytes)).Trace("send msg")
	return p2p.Send(p.rw, msgType.Code(), msgBytes)

}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendMessages(messages []*p2PMessage) {
	select {
	case p.queuedMsg <- messages:
		for _, msg := range messages {
			key := msg.msgKey()
			p.knownMsg.Add(key)
		}
	default:
		msgLog.WithField("count", len(messages)).Debug("Dropping transaction propagation")
	}
}

func (p *peer) AsyncSendMessage(msg *p2PMessage) {
	var messages []*p2PMessage
	messages = append(messages, msg)
	select {
	case p.queuedMsg <- messages:
		key := msg.msgKey()
		p.knownMsg.Add(key)
	default:
		msgLog.Debug("Dropping transaction propagation")
	}
}

// SendNodeData sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data []byte) error {
	return p.sendRawMessage(NodeDataMsg, data)
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes types.Hashes) error {
	msgLog.WithField("count", len(hashes)).Debug("Fetching batch of state data")
	hashsStruct := types.Hashes(hashes)
	b, _ := hashsStruct.MarshalMsg(nil)
	return p.sendRawMessage(GetNodeDataMsg, b)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes types.Hashes) error {
	msgLog.WithField("count", len(hashes)).Debug("Fetching batch of receipts")
	b, _ := hashes.MarshalMsg(nil)
	return p.sendRawMessage(GetReceiptsMsg, b)
}

// RequestHeadersByHash fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestTxsByHash(seqHash types.Hash, seqId uint64) error {
	hash := seqHash
	msg := &types.MessageTxsRequest{
		SeqHash:   &hash,
		Id:        &seqId,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeTxsRequest, msg)
}

func (p *peer) RequestTxs(hashs types.Hashes) error {
	msg := &types.MessageTxsRequest{
		Hashes:    &hashs,
		RequestId: MsgCounter.Get(),
	}

	return p.sendRequest(MessageTypeTxsRequest, msg)
}

func (p *peer) RequestTxsById(seqId uint64) error {
	msg := &types.MessageTxsRequest{
		Id:        &seqId,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeTxsRequest, msg)
}

func (p *peer) RequestBodies(seqHashs types.Hashes) error {
	msg := &types.MessageBodiesRequest{
		SeqHashes: seqHashs,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeBodiesRequest, msg)
}

func (h *Hub) RequestOneHeader(peerId string, hash types.Hash) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.RequestOneHeader(hash)
}

func (h *Hub) RequestBodies(peerId string, hashs types.Hashes) error {
	p := h.peers.Peer(peerId)
	if p == nil {
		return fmt.Errorf("peer not found")
	}
	return p.RequestBodies(hashs)
}

func (p *peer) RequestOneHeader(hash types.Hash) error {
	tmpHash := hash
	msg := &types.MessageHeaderRequest{
		Origin: types.HashOrNumber{
			Hash: &tmpHash,
		},
		Amount:    uint64(1),
		Skip:      uint64(0),
		Reverse:   false,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeHeaderRequest, msg)
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (p *peer) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	msg := &types.MessageHeaderRequest{
		Origin: types.HashOrNumber{
			Number: &origin,
		},
		Amount:    uint64(amount),
		Skip:      uint64(skip),
		Reverse:   reverse,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeHeaderRequest, msg)
}

func (p *peer) RequestHeadersByHash(hash types.Hash, amount int, skip int, reverse bool) error {
	tmpHash := hash
	msg := &types.MessageHeaderRequest{
		Origin: types.HashOrNumber{
			Hash: &tmpHash,
		},
		Amount:    uint64(amount),
		Skip:      uint64(skip),
		Reverse:   reverse,
		RequestId: MsgCounter.Get(),
	}
	return p.sendRequest(MessageTypeHeaderRequest, msg)
}

func (p *peer) sendRequest(msgType MessageType, request types.Message) error {
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
	goroutine.NewRoutine( p.broadcast)

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
	indices := generateRandomIndices(n, len(all))
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
	indices := generateRandomIndices(n, len(all))
	for _, i := range indices {
		list = append(list, all[i])
	}
	return list
}

// generate [count] unique random numbers within range [0, upper)
// if count > upper, use all available indices
func generateRandomIndices(count int, upper int) []int {
	if count > upper {
		count = upper
	}
	// avoid dup
	generated := make(map[int]struct{})
	for count > len(generated) {
		i := rand.Intn(upper)
		generated[i] = struct{}{}
	}
	arr := make([]int, 0, len(generated))
	for k := range generated {
		arr = append(arr, k)
	}
	return arr
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutMsg(hash types.Hash, m MessageType) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	if m == MessageTypeNewTx || m == MessageTypeNewSequencer {
		keyControl := newMsgKey(MessageTypeControl, hash)
		key := newMsgKey(m, hash)
		for _, p := range ps.peers {
			if !p.knownMsg.Contains(key) && !p.knownMsg.Contains(keyControl) {
				list = append(list, p)
			}
		}
		return list
	}
	key := newMsgKey(m, hash)
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

// Handshake executes the og protocol handshake, negotiating version number,
// network IDs, head and genesis blocks.
func (p *peer) Handshake(network uint64, head types.Hash, seqId uint64, genesis types.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status StatusData // safe to read after two values have been received from errc

	sendStatusFunc:= func()() {
		s := StatusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			CurrentBlock:    head,
			CurrentId:       seqId,
			GenesisBlock:    genesis,
		}
		data, _ := s.MarshalMsg(nil)
		errc <- p2p.Send(p.rw, p2p.MsgCodeType(StatusMsg), data)
	}
	goroutine.NewRoutine(sendStatusFunc)
	readFunc :=func()() {
		errc <- p.readStatus(network, &status, genesis)
	}
	goroutine.NewRoutine( readFunc)
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

func (p *peer) readStatus(network uint64, status *StatusData, genesis types.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != p2p.MsgCodeType(StatusMsg) {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	data, err := msg.GetPayLoad()
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	_, err = status.UnmarshalMsg(data)
	// Decode the handshake and make sure everything matches
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock.Bytes[:8], genesis.Bytes[:8])
	}
	if status.NetworkId != network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}
