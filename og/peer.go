package og

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/types"
	mapset "github.com/deckarep/golang-set"
	log "github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
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
	td        *big.Int
	lock      sync.RWMutex
	knownMsg  mapset.Set       // Set of transaction hashes known to be known by this peer
	queuedMsg chan []*P2PMessage // Queue of transactions to broadcast to the peer
	term      chan struct{}    // Termination channel to stop the broadcaster
}

type PeerInfo struct {
	Version    int      `json:"version"`    // Ethereum protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:      p,
		rw:        rw,
		version:   version,
		id:        fmt.Sprintf("%x", p.ID().Bytes()[:8]),
		knownMsg:  mapset.NewSet(),
		queuedMsg: make(chan []*types.Tx, maxqueuedMsg), // TODO: compile error
		term:      make(chan struct{}),
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
				return
			}
			log.Debug("Broadcast transactions", "count", len(msg))

		case <-p.term:
			return
		}
	}
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash := p.Head()

	return &PeerInfo{
		Version: p.version,
		Head:    hash.Hex(),
	}
}

// Head retrieves a copy of the current head hash and total difficulty of the
// peer.
func (p *peer) Head() (hash types.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash.Bytes[:], p.head.Bytes[:])
	return hash
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash types.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head.Bytes[:], hash.Bytes[:])
	p.td.Set(td)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash types.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownMsg.Cardinality() >= maxknownMsg {
		p.knownMsg.Pop()
	}
	p.knownMsg.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendMessages(messages []*P2PMessage) error {
	var msgBytes []byte
	for _, msg := range messages {
		p.knownMsg.Add(msg.Hash)
		msgBytes= append(msgBytes,msg.Message...)
	}
	return p2p.Send(p.rw, TxMsg, msgBytes)
}

// AsyncSendTransactions queues list of transactions propagation to a remote
// peer. If the peer's broadcast queue is full, the event is silently dropped.
func (p *peer) AsyncSendMessages(messages []*P2PMessage) {
	select {
	case p.queuedMsg <- messages:
		for _, tx := range messages {
			p.knownMsg.Add(tx.Hash)
		}
	default:
		log.Debug("Dropping transaction propagation", "count", len(messages))
	}
}

func (p *peer) AsyncSendMessage(message *P2PMessage) {
	var messages []*P2PMessage
	messages = append(messages,message)
	select {
	case p.queuedMsg <- messages:
			p.knownMsg.Add(message.Hash)
	default:
		log.Debug("Dropping transaction propagation")
	}
}
// SendNodeDataRLP sends a batch of arbitrary internal data, corresponding to the
// hashes requested.
func (p *peer) SendNodeData(data []byte) error {
	return p2p.Send(p.rw, NodeDataMsg, data)
}

// RequestBodies fetches a batch of blocks' bodies corresponding to the hashes
// specified.
func (p *peer) RequestTransactions(hashes types.Hashs) error {
	log.Debug("Fetching batch of block bodies", "count", len(hashes))
	b,_:= hashes.MarshalMsg(nil)
	return p2p.Send(p.rw, GetTxMsg, b)
}

// RequestNodeData fetches a batch of arbitrary data from a node's known state
// data, corresponding to the specified hashes.
func (p *peer) RequestNodeData(hashes types.Hashs) error {
	log.Debug("Fetching batch of state data", "count", len(hashes))
	b,_:= hashes.MarshalMsg(nil)
	return p2p.Send(p.rw, GetNodeDataMsg, b)
}

// RequestReceipts fetches a batch of transaction receipts from a remote node.
func (p *peer) RequestReceipts(hashes types.Hashs) error {
	log.Debug("Fetching batch of receipts", "count", len(hashes))
	b,_:= hashes.MarshalMsg(nil)
	return p2p.Send(p.rw, GetReceiptsMsg, b)
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("eth/%2d", p.version),
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
	go p.broadcast()

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

func (ps *peerSet)Peers()([]*peer) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		list = append(list, p)
	}
	return list
}


// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash types.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownMsg.Contains(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var bestPeer *peer

	for _, p := range ps.peers {
		//todo
		bestPeer = p
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
