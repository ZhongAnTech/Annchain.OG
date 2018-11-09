package og

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/types"
	"sync"
	"time"
)

// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandler struct {
	Og           *Og
	Hub          *Hub
	requestCache *Cache
}

//msg request cache ,don't send duplicate message
type Cache struct {
	cache map[uint64]bool
	mu    sync.RWMutex
}

//NewIncomingMessageHandler
func NewIncomingMessageHandler(og *Og, hub *Hub) *IncomingMessageHandler {
	return &IncomingMessageHandler{
		Og:  og,
		Hub: hub,
		requestCache: &Cache{
			cache: make(map[uint64]bool),
		},
	}
}

func (h *IncomingMessageHandler) HandleFetchByHashRequest(syncRequest types.MessageSyncRequest, peerId string) {
	if len(syncRequest.Hashes) == 0 {
		msgLog.Debug("empty MessageSyncRequest")
		return
	}

	var txs []*types.Tx
	var seqs []*types.Sequencer

	for _, hash := range syncRequest.Hashes {
		txi := h.Og.TxPool.Get(hash)
		if txi == nil {
			txi = h.Og.Dag.GetTx(hash)
		}
		switch tx := txi.(type) {
		case *types.Sequencer:
			seqs = append(seqs, tx)
		case *types.Tx:
			txs = append(txs, tx)
		}

	}
	syncResponse := types.MessageSyncResponse{
		Txs:        txs,
		Sequencers: seqs,
	}
	h.Hub.SendToPeer(peerId, MessageTypeFetchByHashResponse, &syncResponse)
}

func (h *IncomingMessageHandler) HandleHeaderResponse(headerMsg types.MessageHeaderResponse, peerId string) {

	headers := headerMsg.Sequencers
	// Filter out any explicitly requested headers, deliver the rest to the downloader
	seqHeaders := types.SeqsToHeaders(headers)
	filter := len(seqHeaders) == 1

	// TODO: verify fetcher
	if filter {
		// Irrelevant of the fork checks, send the header to the fetcher just in case
		seqHeaders = h.Hub.Fetcher.FilterHeaders(peerId, seqHeaders, time.Now())
	}
	if len(seqHeaders) > 0 || !filter {
		err := h.Hub.Downloader.DeliverHeaders(peerId, seqHeaders)
		if err != nil {
			msgLog.WithError(err).Debug("Failed to deliver headers")
		}
	}
	msgLog.WithField("header lens", len(seqHeaders)).Trace("handle MessageTypeHeaderResponse")
}

func (h *IncomingMessageHandler) HandleHeaderRequest(query types.MessageHeaderRequest, peerId string) {
	hashMode := !query.Origin.Hash.Empty()
	first := true
	msgLog.WithField("hash", query.Origin.Hash).WithField("number", query.Origin.Number).WithField(
		"hashmode", hashMode).WithField("amount", query.Amount).WithField("skip", query.Skip).Trace("requests")
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
				origin = h.Og.Dag.GetSequencerByHash(query.Origin.Hash)
				if origin != nil {
					query.Origin.Number = origin.Number()
				}
			} else {
				origin = h.Og.Dag.GetSequencer(query.Origin.Hash, query.Origin.Number)
			}
		} else {
			origin = h.Og.Dag.GetSequencerById(query.Origin.Number)
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
				seq := h.Og.Dag.GetSequencerById(query.Origin.Number - ancestor)
				query.Origin.Hash, query.Origin.Number = seq.GetTxHash(), seq.Number()
				unknown = query.Origin.Hash.Empty()
			}
		case hashMode && !query.Reverse:
			// Hash based traversal towards the leaf block
			var (
				current = origin.Number()
				next    = current + query.Skip + 1
			)
			if next <= current {
				msgLog.Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", peerId)
				unknown = true
			} else {
				if header := h.Og.Dag.GetSequencerById(next); header != nil {
					nextHash := header.GetTxHash()
					oldSeq := h.Og.Dag.GetSequencerById(next - (query.Skip + 1))
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
		Sequencers:  headers,
		RequestedId: query.RequestId,
	}
	h.Hub.SendToPeer(peerId, MessageTypeHeaderResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleTxsResponse(request types.MessageTxsResponse) {
	if request.Sequencer != nil {
		msgLog.WithField("len", len(request.Txs)).WithField("seq id", request.Sequencer.Id).Trace("got response txs ")
	} else {
		msgLog.Warn("got nil sequencer")
		return
	}

	lseq := h.Og.Dag.LatestSequencer()
	//todo need more condition
	if lseq.Number() < request.Sequencer.Number() {
		h.Og.TxBuffer.AddRemoteTxs(request.Sequencer, request.Txs)
	}
	return
}

func (h *IncomingMessageHandler) HandleTxsRequest(msgReq types.MessageTxsRequest, peerId string) {
	var msgRes types.MessageTxsResponse

	var seq *types.Sequencer
	if msgReq.SeqHash != nil && msgReq.Id != 0 {
		seq = h.Og.Dag.GetSequencer(*msgReq.SeqHash, msgReq.Id)
	} else {
		seq = h.Og.Dag.GetSequencerById(msgReq.Id)
	}
	msgRes.Sequencer = seq
	if seq != nil {
		msgRes.Txs = h.Og.Dag.GetTxsByNumber(seq.Id)
	} else {
		msgLog.WithField("id", msgReq.Id).WithField("hash", msgReq.SeqHash).Warn("seq was not found for request ")
	}
	h.Hub.SendToPeer(peerId, MessageTypeTxsResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleBodiesResponse(request types.MessageBodiesResponse, peerId string) {
	// Deliver them all to the downloader for queuing
	transactions := make([][]*types.Tx, len(request.Bodies))
	sequencers := make([]*types.Sequencer, len(request.Bodies))
	for i, bodyData := range request.Bodies {
		var body types.MessageTxsResponse
		_, err := body.UnmarshalMsg(bodyData)
		if err != nil {
			msgLog.WithError(err).Warn("decode error")
			break
		}
		if body.Sequencer == nil {
			msgLog.Warn(" body.Sequencer is nil")
			break
		}
		transactions[i] = body.Txs
		sequencers[i] = body.Sequencer
	}
	msgLog.WithField("bodies len", len(request.Bodies)).Trace("got bodies")

	// Filter out any explicitly requested bodies, deliver the rest to the downloader
	filter := len(transactions) > 0 || len(sequencers) > 0
	// TODO: verify fetcher
	if filter {
		transactions = h.Hub.Fetcher.FilterBodies(peerId, transactions, sequencers, time.Now())
	}
	if len(transactions) > 0 || len(sequencers) > 0 || !filter {
		msgLog.WithField("txs len", len(transactions)).WithField("seq len", len(sequencers)).Trace("deliver bodies ")
		err := h.Hub.Downloader.DeliverBodies(peerId, transactions, sequencers)
		if err != nil {
			msgLog.Debug("Failed to deliver bodies", "err", err)
		}
	}
	msgLog.Debug("handle MessageTypeBodiesResponse")
	return
}

func (h *IncomingMessageHandler) HandleBodiesRequest(msgReq types.MessageBodiesRequest, peerId string) {
	var msgRes types.MessageBodiesResponse
	var bytes int

	for i := 0; i < len(msgReq.SeqHashes); i++ {
		seq := h.Og.Dag.GetSequencerByHash(msgReq.SeqHashes[i])
		if seq == nil {
			msgLog.WithField("hash", msgReq.SeqHashes[i]).Warn("seq is nil")
			break
		}
		if bytes >= softResponseLimit {
			msgLog.Debug("reached softResponseLimit ")
			break
		}
		if len(msgRes.Bodies) >= downloader.MaxBlockFetch {
			msgLog.Debug("reached MaxBlockFetch 128 ")
			break
		}
		var body types.MessageTxsResponse
		body.Sequencer = seq
		body.Txs = h.Og.Dag.GetTxsByNumber(seq.Id)
		bodyData, _ := body.MarshalMsg(nil)
		bytes += len(bodyData)
		msgRes.Bodies = append(msgRes.Bodies, types.RawData(bodyData))
	}
	msgRes.RequestedId = msgReq.RequestId
	h.Hub.SendToPeer(peerId, MessageTypeBodiesResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleSequencerHeader(msgHeader types.MessageSequencerHeader, peerId string) {
	if msgHeader.Hash == nil {
		return
	}

	//no need to broadcast again ,just all our peers need know this ,not all network
	//set peer's head
	h.Hub.SetPeerHead(peerId, *msgHeader.Hash, msgHeader.Number)

	//if h.SyncManager.Status != syncer.SyncStatusIncremental{
	//	return
	//}
	return
	// TODO:
	lseq := h.Og.Dag.LatestSequencer()
	if msgHeader.Number > lseq.Number() {
		if !h.requestCache.get(msgHeader.Number) {
			h.Hub.Fetcher.Notify(peerId, *msgHeader.Hash, msgHeader.Number, time.Now(), h.Hub.RequestOneHeader, h.Hub.RequestBodies)
			h.requestCache.add(msgHeader.Number)
			msgLog.WithField("header ", msgHeader.String()).Info("notify to header to fetcher")
		}
		h.requestCache.clean(lseq.Number())
	}

	return
}

func (c *Cache) add(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[id] = true
}

func (c *Cache) get(id uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[id]
}

func (c *Cache) remove(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, id)
}

func (c *Cache) removeItems(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, id)
}

func (c *Cache) clean(lseqId uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, _ := range c.cache {
		if k <= lseqId {
			delete(c.cache, k)
		}
	}

}

func (h *IncomingMessageHandler) HandlePing(peerId string) {
	msgLog.Debug("received your ping. Respond you a pong")
	h.Hub.SendBytesToPeer(peerId, MessageTypePong, []byte{1})
}

func (h *IncomingMessageHandler) HandlePong() {
	msgLog.Debug("received your pong.")
}
