package og

import (
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

const (
	forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing

	// This is the target size for the packs of transactions sent by txsyncLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txsyncPackSize = 100 * 1024
)

type txsync struct {
	p   *peer
	txs []*types.Tx
}

func (h *Hub) GetPendingTxs() types.Txs {
	return nil
}

// syncTransactions starts sending all currently pending transactions to the given peer.
func (h *Hub) syncTransactions(p *peer) {
	var txs types.Txs
	txs = h.GetPendingTxs()
	if len(txs) == 0 {
		return
	}
	select {
	case h.txsyncCh <- &txsync{p, txs}:
	case <-h.quitSync:
	}
}

// txsyncLoop takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (h *Hub) txsyncLoop() {
	var (
		pending = make(map[discover.NodeID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)

	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		// Fill pack with transactions up to the target size.
		size := common.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += common.StorageSize(float64(s.txs[i].Msgsize()))
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		log.Debug("Sending batch of transactions", "count", len(pack.txs), "bytes", size)
		sending = true
		go func() { done <- pack.p.SendTransactions(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-h.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				log.Debug("Transaction send failed", "err", err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-h.quitSync:
			return
		}
	}
}

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (h *Hub) syncer() {
	// Start and ensure cleanup of sync mechanisms
	h.fetcher.Start()
	defer h.fetcher.Stop()
	defer h.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-h.newPeerCh:
			// Make sure we have peers to select from, then sync
			if h.peers.Len() < minDesiredPeerCount {
				break
			}
			go h.synchronise(h.peers.BestPeer())

		case <-forceSync.C:
			// Force a sync even if not enough peers are present
			go h.synchronise(h.peers.BestPeer())

		case <-h.noMorePeers:
			log.Info("got quit message ,quit hub syncer")
			return
		}
	}
}

// synchronise tries to sync up our local block chain with a remote peer.
func (h *Hub) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}
	// Make sure the peer's TD is higher than our own
	currentBlock := h.Dag.LatestSequencer()
	seqId := currentBlock.Number()

	pHead, pSeqid := peer.Head()
	if seqId >= pSeqid {
		return
	}
	// Otherwise try to sync with the downloader
	mode := downloader.FullSync
	if atomic.LoadUint32(&h.fastSync) == 1 {
		// Fast sync was explicitly requested, and explicitly granted
		mode = downloader.FastSync
	} else if currentBlock.Number() == 0 {
		// The database seems empty as the current block is the genesis. Yet the fast
		// block is ahead, so fast sync was enabled for this node at a certain point.
		// The only scenario where this can happen is if the user manually (or via a
		// bad block) rolled back a fast sync node below the sync point. In this case
		// however it's safe to reenable fast sync.
		//todo
		//atomic.StoreUint32(&h.fastSync, 1)
		//mode = downloader.FastSync
	}
	log.Debug("sync with best peer   ", pHead)
	log.WithField("our id", seqId).WithField(" peer id ", pSeqid).Debug("sync with")
	// Run the sync cycle, and disable fast sync if we've went past the pivot block
	if err := h.downloader.Synchronise(peer.id, pHead, pSeqid, mode); err != nil {
		log.WithError(err).Warn("sync failed")
		return
	}
	if atomic.LoadUint32(&h.fastSync) == 1 {
		log.Info("Fast sync complete, auto disabling")
		atomic.StoreUint32(&h.fastSync, 0)
	}
	atomic.StoreUint32(&h.acceptTxs, 1) // Mark initial sync done
	if head := h.Dag.LatestSequencer(); head.Number() > 0 {
		// We've completed a sync cycle, notify all peers of new state. This path is
		// essential in star-topology networks where a gateway node needs to notify
		// all its out-of-date peers of the availability of a new block. This failure
		// scenario will most often crop up in private and hackathon networks with
		// degenerate connectivity, but it should be healthy for the mainnet too to
		// more reliably update peers or the local TD state.
		//go h.BroadcastBlock(head, false)
		data, _ := head.MarshalMsg(nil)
		h.SendMessage(MessageTypeNewSequence, data)
	}
}
