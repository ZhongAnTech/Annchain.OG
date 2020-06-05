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
	"fmt"
	"github.com/annchain/OG/arefactor/common/goroutine"
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/consensus/campaign"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/og/types"
	"sort"
	"sync/atomic"

	// "github.com/annchain/OG/ffchan"
	"sync"
	"time"

	"github.com/annchain/OG/og/downloader"
)

// IncomingMessageHandler is the default handler of all incoming messages for OG
type IncomingMessageHandler struct {
	Og              *Og
	Hub             *Hub
	controlMsgCache *ControlMsgCache
	requestCache    *RequestCache
	TxEnable        func() bool
	quit            chan struct{}
}

type hashAndSourceId struct {
	hash     types2.Hash
	sourceId string
}

//msg request cache ,don't send duplicate Message
type ControlMsgCache struct {
	cache      map[types2.Hash]*controlItem
	mu         sync.RWMutex
	size       int
	queue      chan *hashAndSourceId
	ExpireTime time.Duration
}

type controlItem struct {
	sourceId   string
	receivedAt *time.Time
	requested  bool
}

type RequestCache struct {
	cache map[uint64]bool
	mu    sync.RWMutex
}

//NewIncomingMessageHandler
func NewIncomingMessageHandler(og *Og, hub *Hub, cacheSize int, expireTime time.Duration) *IncomingMessageHandler {
	return &IncomingMessageHandler{
		Og:  og,
		Hub: hub,
		controlMsgCache: &ControlMsgCache{
			cache:      make(map[types2.Hash]*controlItem),
			size:       cacheSize,
			ExpireTime: expireTime,
			queue:      make(chan *hashAndSourceId, 1),
		},
		requestCache: &RequestCache{
			cache: make(map[uint64]bool),
		},

		quit: make(chan struct{}),
	}
}

func (h *IncomingMessageHandler) HandleFetchByHashRequest(syncRequest *p2p_message.MessageSyncRequest, peerId string) {
	var txs types.TxisMarshaler
	//var index []uint32
	//encode bloom filter , send txs that the peer dose't have
	if syncRequest.Filter != nil && len(syncRequest.Filter.Data) > 0 {
		err := syncRequest.Filter.Decode()
		if err != nil {
			message_archive.msgLog.WithError(err).Warn("encode bloom filter error")
			return
		}
		if syncRequest.Height == nil {
			message_archive.msgLog.WithError(err).Warn("param error, height is nil")
			return
		}
		height := *syncRequest.Height
		ourHeight := h.Og.Dag.LatestSequencer().Number()
		if height < ourHeight {
			message_archive.msgLog.WithField("ourHeight ", ourHeight).WithField("height", height).Warn("our height is smaller")
			return
		} else {
			var filterHashes types2.Hashes
			if height == ourHeight {
				filterHashes = h.Og.TxPool.GetHashOrder()
			} else if height < ourHeight {
				dagHashes := h.Og.Dag.GetTxsHashesByNumber(height + 1)
				if dagHashes != nil {
					filterHashes = *dagHashes
				}
				filterHashes = append(filterHashes, h.Og.Dag.LatestSequencer().GetHash())
			}
			message_archive.msgLog.WithField("len ", len(filterHashes)).Trace("get hashes")
			for _, hash := range filterHashes {
				ok, err := syncRequest.Filter.LookUpItem(hash.Bytes[:])
				if err != nil {
					message_archive.msgLog.WithError(err).Warn("lookup bloom filter error")
					continue
				}
				//if peer miss this tx ,send it
				if !ok {
					txi := h.Og.TxPool.Get(hash)
					if txi == nil {
						txi = h.Og.Dag.GetTx(hash)
					}
					txs.Append(txi)
				}
			}

			// uint64(0) -2 >0
			if height+2 <= ourHeight {
				dagTxs := h.Og.Dag.GetTxisByNumber(height + 2)
				rtxs := types.NewTxisMarshaler(dagTxs)
				if rtxs != nil && len(rtxs) != 0 {
					txs = append(txs, rtxs...)
				}
				//index = append(index, uint32(len(txs)))
				seq := h.Og.Dag.GetSequencerByHeight(height + 2)
				txs.Append(seq)
			}
			message_archive.msgLog.WithField("to ", peerId).WithField("to request ", syncRequest.RequestId).WithField("len txs ", len(txs)).Debug("will send txs after bloom filter")
		}
	} else if syncRequest.Hashes != nil && len(*syncRequest.Hashes) > 0 {
		for _, hash := range *syncRequest.Hashes {
			txi := h.Og.TxPool.Get(hash)
			if txi == nil {
				txi = h.Og.Dag.GetTx(hash)
			}
			if txi == nil {
				continue
			}
			txs.Append(txi)
		}
	} else if syncRequest.HashTerminats != nil {
		hashMap := make(map[p2p_message.HashTerminat]int)
		allhashs := h.Og.TxPool.GetOrder()
		if len(*syncRequest.HashTerminats) > 0 {
			//var hashTerminates p2p_message.HashTerminats
			for i, hash := range allhashs {
				var hashTerminate p2p_message.HashTerminat
				copy(hashTerminate[:], hash.Bytes[:4])
				//hashTerminates = append(hashTerminates, hashTerminate)
				hashMap[hashTerminate] = i
			}
			for _, hash := range *syncRequest.HashTerminats {
				if _, ok := hashMap[hash]; ok {
					delete(hashMap, hash)
				}
			}
			for _, v := range hashMap {
				hash := allhashs[v]
				//if peer miss this tx ,send it
				txi := h.Og.TxPool.Get(hash)
				if txi == nil {
					txi = h.Og.Dag.GetTx(hash)
				}
				txs.Append(txi)
			}
			message_archive.msgLog.WithField("your tx num", len(*syncRequest.HashTerminats)).WithField(
				"our tx num ", len(allhashs)).WithField("response tx len", len(txs)).WithField(
				"to ", peerId).Debug("response to hashList")
		} else {
			for _, hash := range allhashs {
				//if peer miss this tx ,send it
				txi := h.Og.TxPool.Get(hash)
				if txi == nil {
					txi = h.Og.Dag.GetTx(hash)
				}
				txs.Append(txi)
			}
		}
	} else {
		message_archive.msgLog.Debug("empty MessageBatchSyncRequest")
		return
	}
	if len(txs) > 0 {
		msgRes := p2p_message.MessageSyncResponse{
			RawTxs: &txs,
			//SequencerIndex: index,
			RequestedId: syncRequest.RequestId,
		}
		if txs != nil && len(txs) != 0 {
			msgRes.RawTxs = &txs
		}
		h.Hub.SendToPeer(peerId, message_archive.MessageTypeFetchByHashResponse, &msgRes)
	} else {
		message_archive.msgLog.Debug("empty Data , did't send")
	}
	return
}

func (h *IncomingMessageHandler) HandleHeaderResponse(headerMsg *p2p_message.MessageHeaderResponse, peerId string) {

	// Filter out any explicitly requested headers, deliver the rest to the downloader
	if headerMsg.Headers == nil {
		message_archive.msgLog.Warn("nil MessageHeaderResponse headers")
		return
	}

	seqHeaders := *headerMsg.Headers
	filter := len(seqHeaders) == 1

	// TODO: verify fetcher
	if filter {
		// Irrelevant of the fork checks, send the header to the fetcher just in case
		seqHeaders = h.Hub.Fetcher.FilterHeaders(peerId, seqHeaders, time.Now())
	}
	if len(seqHeaders) > 0 || !filter {
		err := h.Hub.Downloader.DeliverHeaders(peerId, seqHeaders)
		if err != nil {
			message_archive.msgLog.WithError(err).Debug("Failed to deliver headers")
		}
	}
	message_archive.msgLog.WithField("headers", headerMsg).WithField("header lens", len(seqHeaders)).Debug("handle p2p_message.MessageTypeHeaderResponse")
}

func (h *IncomingMessageHandler) HandleHeaderRequest(query *p2p_message.MessageHeaderRequest, peerId string) {
	hashMode := query.Origin.Hash != nil
	if query.Origin.Number == nil {
		i := uint64(0)
		query.Origin.Number = &i
	}
	first := true
	message_archive.msgLog.WithField("Hash", query.Origin.Hash).WithField("number", query.Origin.Number).WithField(
		"hashmode", hashMode).WithField("amount", query.Amount).WithField("skip", query.Skip).Trace("requests")
	// Gather headers until the fetch or network limits is reached
	var (
		bytes   common.StorageSize
		headers archive.Sequencers
		unknown bool
	)
	for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
		// Retrieve the next header satisfying the query
		var origin *types.Sequencer
		if hashMode {
			if first {
				first = false
				origin = h.Og.Dag.GetSequencerByHash(*query.Origin.Hash)
				if origin != nil {
					numBer := origin.Number()
					query.Origin.Number = &numBer
				}
			} else {
				origin = h.Og.Dag.GetSequencer(*query.Origin.Hash, *query.Origin.Number)
			}
		} else {
			origin = h.Og.Dag.GetSequencerByHeight(*query.Origin.Number)
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
				seq := h.Og.Dag.GetSequencerByHeight(*query.Origin.Number - ancestor)
				hash := seq.GetHash()
				num := seq.Number()
				query.Origin.Hash, query.Origin.Number = &hash, &num
				unknown = query.Origin.Hash == nil
			}
		case hashMode && !query.Reverse:
			// Hash based traversal towards the leaf block
			var (
				current = origin.Number()
				next    = current + query.Skip + 1
			)
			if next <= current {
				message_archive.msgLog.Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", peerId)
				unknown = true
			} else {
				if header := h.Og.Dag.GetSequencerByHeight(next); header != nil {
					nextHash := header.GetHash()
					oldSeq := h.Og.Dag.GetSequencerByHeight(next - (query.Skip + 1))
					expOldHash := oldSeq.GetHash()
					if expOldHash == *query.Origin.Hash {
						num := next
						query.Origin.Hash, query.Origin.Number = &nextHash, &num
					} else {
						unknown = true
					}
				} else {
					unknown = true
				}
			}
		case query.Reverse:
			// Number based traversal towards the genesis block
			if *query.Origin.Number >= query.Skip+1 {
				*query.Origin.Number -= query.Skip + 1
			} else {
				unknown = true
			}

		case !query.Reverse:
			// Number based traversal towards the leaf block
			*query.Origin.Number += query.Skip + 1
		}
	}
	headres := headers.ToHeaders()
	msgRes := p2p_message.MessageHeaderResponse{
		Headers:     &headres,
		RequestedId: query.RequestId,
	}
	h.Hub.SendToPeer(peerId, message_archive.MessageTypeHeaderResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleTxsResponse(request *p2p_message.MessageTxsResponse) {
	var rawTxs types.TxisMarshaler
	var txis types.Txis
	if request.RawTxs != nil {
		rawTxs = *request.RawTxs
	}
	if request.RawSequencer != nil {
		message_archive.msgLog.WithField("len rawTx", len(rawTxs)).WithField("seq height", request.RawSequencer.Height).Trace(
			"got response txs")
	} else {
		message_archive.msgLog.Warn("got nil sequencer")
		return
	}
	seq := request.RawSequencer.Sequencer()
	lseq := h.Og.Dag.LatestSequencer()
	//todo need more condition
	if lseq.Number() < seq.Number() {
		h.Og.TxBuffer.ReceivedNewTxChan <- seq
		// <-ffchan.NewTimeoutSenderShort(h.Og.TxBuffer.ReceivedNewTxChan, seq, "HandleTxsResponse").C
		txis = rawTxs.Txis()
		sort.Sort(txis)
		for _, tx := range txis {
			//todo add to txcache first
			h.Og.TxBuffer.ReceivedNewTxChan <- tx
			// <-ffchan.NewTimeoutSenderShort(h.Og.TxBuffer.ReceivedNewTxChan, tx, "HandleTxsResponse").C
		}
	}
	return
}

func (h *IncomingMessageHandler) HandleTxsRequest(msgReq *p2p_message.MessageTxsRequest, peerId string) {
	var msgRes p2p_message.MessageTxsResponse
	var seq *types.Sequencer
	if msgReq.Id == nil {
		i := uint64(0)
		msgReq.Id = &i
	}
	if msgReq.SeqHash != nil && *msgReq.Id != 0 {
		seq = h.Og.Dag.GetSequencer(*msgReq.SeqHash, *msgReq.Id)
	} else {
		seq = h.Og.Dag.GetSequencerByHeight(*msgReq.Id)
	}
	msgRes.RawSequencer = seq.RawSequencer()
	if seq != nil {
		txs := h.Og.Dag.GetTxisByNumber(seq.Height)
		rtxs := types.NewTxisMarshaler(txs)
		if rtxs != nil && len(rtxs) != 0 {
			msgRes.RawTxs = &rtxs
		}

	} else {
		message_archive.msgLog.WithField("id", msgReq.Id).WithField("Hash", msgReq.SeqHash).Warn("seq was not found for request")
	}
	h.Hub.SendToPeer(peerId, message_archive.MessageTypeTxsResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleBodiesResponse(request *p2p_message.MessageBodiesResponse, peerId string) {
	// Deliver them all to the downloader for queuing
	transactions := make([]types.Txis, len(request.Bodies))
	sequencers := make([]*types.Sequencer, len(request.Bodies))
	for i, bodyData := range request.Bodies {
		var body p2p_message.MessageBodyData
		_, err := body.UnmarshalMsg(bodyData)
		if err != nil {
			message_archive.msgLog.WithError(err).Warn("decode error")
			break
		}
		if body.RawSequencer == nil {
			message_archive.msgLog.Warn(" body.Sequencer is nil")
			break
		}
		txis := body.ToTxis()
		sort.Sort(txis)
		transactions[i] = txis
		sequencers[i] = body.RawSequencer.Sequencer()
	}
	message_archive.msgLog.WithField("bodies len", len(request.Bodies)).Trace("got bodies")

	// Filter out any explicitly requested bodies, deliver the rest to the downloader
	filter := len(transactions) > 0 || len(sequencers) > 0
	// TODO: verify fetcher
	if filter {
		transactions = h.Hub.Fetcher.FilterBodies(peerId, transactions, sequencers, time.Now())
	}
	if len(transactions) > 0 || len(sequencers) > 0 || !filter {
		message_archive.msgLog.WithField("txs len", len(transactions)).WithField("seq len", len(sequencers)).Trace("deliver bodies")
		err := h.Hub.Downloader.DeliverBodies(peerId, transactions, sequencers)
		if err != nil {
			message_archive.msgLog.Debug("Failed to deliver bodies", "err", err)
		}
	}
	message_archive.msgLog.Debug("handle MessageTypeBodiesResponse")
	return
}

func (h *IncomingMessageHandler) HandleBodiesRequest(msgReq *p2p_message.MessageBodiesRequest, peerId string) {
	var msgRes p2p_message.MessageBodiesResponse
	var bytes int

	for i := 0; i < len(msgReq.SeqHashes); i++ {
		seq := h.Og.Dag.GetSequencerByHash(msgReq.SeqHashes[i])
		if seq == nil {
			message_archive.msgLog.WithField("Hash", msgReq.SeqHashes[i]).Warn("seq is nil")
			break
		}
		if bytes >= softResponseLimit {
			message_archive.msgLog.Debug("reached softResponseLimit")
			break
		}
		if len(msgRes.Bodies) >= downloader.MaxBlockFetch {
			message_archive.msgLog.Debug("reached MaxBlockFetch 128")
			break
		}
		var body p2p_message.MessageBodyData
		body.RawSequencer = seq.RawSequencer()
		txs := h.Og.Dag.GetTxisByNumber(seq.Height)
		rtxs := types.NewTxisMarshaler(txs)
		if rtxs != nil && len(rtxs) != 0 {
			body.RawTxs = &rtxs
		}
		bodyData, _ := body.MarshalMsg(nil)
		bytes += len(bodyData)
		msgRes.Bodies = append(msgRes.Bodies, p2p_message.RawData(bodyData))
	}
	msgRes.RequestedId = msgReq.RequestId
	h.Hub.SendToPeer(peerId, message_archive.MessageTypeBodiesResponse, &msgRes)
}

func (h *IncomingMessageHandler) HandleSequencerHeader(msgHeader *p2p_message.MessageSequencerHeader, peerId string) {
	if msgHeader.Hash == nil {
		return
	}
	if msgHeader.Number == nil {
		i := uint64(0)
		msgHeader.Number = &i
	}
	number := *msgHeader.Number
	//no need to broadcast again ,just all our peers need know this ,not all network
	//set peer's head
	h.Hub.SetPeerHead(peerId, *msgHeader.Hash, *msgHeader.Number)

	//if h.SyncManager.Status != syncer.SyncStatusIncremental{
	//	return
	//}
	return
	// TODO:
	lseq := h.Og.Dag.LatestSequencer()
	if number > lseq.Number() {
		if !h.requestCache.get(number) {
			h.Hub.Fetcher.Notify(peerId, *msgHeader.Hash, number, time.Now(), h.Hub.RequestOneHeader, h.Hub.RequestBodies)
			h.requestCache.set(number)
			message_archive.msgLog.WithField("header ", msgHeader.String()).Info("notify to header to fetcher")
		}
		h.requestCache.clean(lseq.Number())
	}

	return
}

func (c *RequestCache) set(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[id] = true
}

func (c *RequestCache) get(id uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[id]
}

func (c *RequestCache) remove(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, id)
}

func (c *RequestCache) clean(lseqId uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.cache {
		if k <= lseqId {
			delete(c.cache, k)
		}
	}
}

func (c *ControlMsgCache) set(hash types2.Hash, sourceId string) {
	c.queue <- &hashAndSourceId{hash: hash, sourceId: sourceId}
}

func (c *ControlMsgCache) get(hash types2.Hash) *controlItem {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if v, ok := c.cache[hash]; ok {
		return v
	}
	return nil
}

func (c *ControlMsgCache) getALlKey() types2.Hashes {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var hashes types2.Hashes
	for k := range c.cache {
		hashes = append(hashes, k)
	}
	return hashes
}

func (c *ControlMsgCache) Len() int {
	//c.mu.RLock()
	//defer c.mu.RUnlock()
	return len(c.cache)
}

func (c *ControlMsgCache) remove(hash types2.Hash) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, hash)
}

func (h *IncomingMessageHandler) RemoveControlMsgFromCache(hash types2.Hash) {
	h.controlMsgCache.remove(hash)
}

func (h *IncomingMessageHandler) loop() {
	if h.Hub.broadCastMode != FeedBackMode {
		//TODO
		//return
	}
	c := h.controlMsgCache
	var handling uint32
	for {
		select {
		case h := <-c.queue:
			c.mu.Lock()
			if len(c.cache) > c.size {
				//todo
			}
			now := time.Now()
			item := controlItem{
				receivedAt: &now,
				sourceId:   h.sourceId,
			}
			c.cache[h.hash] = &item
			c.mu.Unlock()
			//todo optimize the time after
		case <-time.After(10 * time.Millisecond):
			if atomic.LoadUint32(&handling) == 1 {
				continue
			}
			atomic.StoreUint32(&handling, 1)
			h.processControlMsg()
			atomic.StoreUint32(&handling, 0)

		case <-h.quit:
			message_archive.msgLog.Info(" incoming msg handler got quit signal ,quiting...")
			return
		}

	}
}

func (h *IncomingMessageHandler) processControlMsg() {
	c := h.controlMsgCache
	keys := c.getALlKey()
	for _, k := range keys {
		item := c.get(k)
		if item == nil {
			continue
		}
		txkey := message_archive.NewMsgKey(message_archive.MessageTypeNewTx, k)
		if _, err := h.Hub.messageCache.GetIFPresent(txkey); err == nil {
			message_archive.msgLog.WithField("Hash ", k).Trace("already received tx of this control msg")
			c.remove(k)
			continue
		}
		if item.receivedAt.Add(2 * time.Millisecond).Before(time.Now()) {
			if h.Hub.IsReceivedHash(k) {
				message_archive.msgLog.WithField("Hash ", k).Trace("already received tx of this control msg")
				c.remove(k)
				continue
			}
			if item.receivedAt.Add(c.ExpireTime).Before(time.Now()) {
				hash := k
				msg := &p2p_message.MessageGetMsg{Hash: &hash}
				message_archive.msgLog.WithField("Hash ", k).Debug("send GetTx msg")
				goroutine.New(func() {
					h.Hub.SendGetMsg(item.sourceId, msg)
				})
				c.remove(k)
			}
		}
	}
}

func (h *IncomingMessageHandler) HandlePing(peerId string) {
	message_archive.msgLog.Debug("received your ping. Respond you a pong")
	h.Hub.SendBytesToPeer(peerId, message_archive.MessageTypePong, []byte{1})
}

func (h *IncomingMessageHandler) HandlePong() {
	message_archive.msgLog.Debug("received your pong.")
}

func (h *IncomingMessageHandler) HandleGetMsg(msg *p2p_message.MessageGetMsg, sourcePeerId string) {
	if msg == nil || msg.Hash == nil {
		message_archive.msgLog.Warn("msg is nil")
		return
	}
	txi := h.Og.TxPool.Get(*msg.Hash)
	if txi == nil {
		txi = h.Og.Dag.GetTx(*msg.Hash)
	}
	if txi == nil {
		message_archive.msgLog.WithField("for Hash ", *msg.Hash).Warn("txi not found")
		return
	}
	switch txi.GetType() {
	case types.TxBaseTypeTx:
		tx := txi.(*types.Tx)
		response := p2p_message.MessageNewTx{RawTx: tx.RawTx()}
		h.Hub.SendToPeer(sourcePeerId, message_archive.MessageTypeNewTx, &response)
	case types.TxBaseTypeTermChange:
		tx := txi.(*campaign.TermChange)
		response := p2p_message.MessageTermChange{RawTermChange: tx.RawTermChange()}
		h.Hub.SendToPeer(sourcePeerId, message_archive.MessageTypeNewTx, &response)
	case types.TxBaseTypeCampaign:
		tx := txi.(*campaign.Campaign)
		response := p2p_message.MessageCampaign{RawCampaign: tx.RawCampaign()}
		h.Hub.SendToPeer(sourcePeerId, message_archive.MessageTypeNewTx, &response)
	case types.TxBaseTypeSequencer:
		tx := txi.(*types.Sequencer)
		response := p2p_message.MessageNewSequencer{RawSequencer: tx.RawSequencer()}
		h.Hub.SendToPeer(sourcePeerId, message_archive.MessageTypeNewSequencer, &response)
	case types.TxBaseAction:
		tx := txi.(*archive.ActionTx)
		response := p2p_message.MessageNewActionTx{ActionTx: tx}
		h.Hub.SendToPeer(sourcePeerId, message_archive.MessageTypeNewSequencer, &response)
	}
	return
}

func (h *IncomingMessageHandler) HandleControlMsg(req *p2p_message.MessageControl, sourceId string) {
	if req.Hash == nil {
		message_archive.msgLog.WithError(fmt.Errorf("miss Hash")).Debug("control msg request err")
		return
	}

	if !h.TxEnable() {
		message_archive.msgLog.Debug("incremental received p2p_message.MessageTypeControl but receiveTx  disabled")
		return
	}
	hash := *req.Hash
	txkey := message_archive.NewMsgKey(message_archive.MessageTypeNewTx, hash)
	if _, err := h.Hub.messageCache.GetIFPresent(txkey); err == nil {
		message_archive.msgLog.WithField("Hash ", hash).Trace("already got tx of this control msg")
		return
	}
	if item := h.controlMsgCache.get(hash); item != nil {
		message_archive.msgLog.WithField("Hash ", hash).Trace("duplicated control msg")
		return
	}
	if h.Hub.IsReceivedHash(hash) {
		message_archive.msgLog.WithField("Hash ", hash).Trace("already received tx of this control msg")
	}
	h.controlMsgCache.set(hash, sourceId)
	message_archive.msgLog.WithField("Hash ", hash).Trace("already received tx of this control msg")
}

func (m *IncomingMessageHandler) Start() {
	goroutine.New(m.loop)
	message_archive.msgLog.Info("Message handler started")
}

func (m *IncomingMessageHandler) Stop() {
	close(m.quit)
	message_archive.msgLog.Info("Message handler stopped")
}

func (m *IncomingMessageHandler) Name() string {
	return "IncomingMessageHandler"
}

func (m *IncomingMessageHandler) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"controlMsgCache": m.controlMsgCache.Len(),
	}
}
