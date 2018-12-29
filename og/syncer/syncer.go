package syncer

import (
	"github.com/annchain/OG/og/txcache"
	"sync"
	"time"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
)

const BloomFilterRate = 5 //sendign 4 req

type MessageSender interface {
	BroadcastMessage(messageType og.MessageType, message types.Message)
	MulticastMessage(messageType og.MessageType, message types.Message)
	MulticastToSource(messageType og.MessageType, message types.Message, sourceMsgHash *types.Hash)
}

type FireHistory struct {
	StartTime  time.Time
	LastTime   time.Time
	FiredTimes int
}

// IncrementalSyncer fetches tx from other  peers. (incremental)
// IncrementalSyncer will not fire duplicate requests in a period of time.
type IncrementalSyncer struct {
	config                  *SyncerConfig
	messageSender           MessageSender
	getTxsHashes            func() []types.Hash
	isKnownHash             func(hash types.Hash) bool
	getHeight               func() uint64
	acquireTxQueue          chan types.Hash
	acquireTxDedupCache     gcache.Cache     // list of hashes that are queried recently. Prevent duplicate requests.
	bufferedIncomingTxCache *txcache.TxCache // cache of incoming txs that are not fired during full sync.
	firedTxCache            gcache.Cache     // cache of hashes that are fired however haven't got any response yet
	quitLoopSync            chan bool
	quitLoopEvent           chan bool
	quitNotifyEvent         chan  bool
	EnableEvent             chan bool
	Enabled                 bool
	OnNewTxiReceived        []chan types.Txi
	notifyTxEvent           chan bool
	notifying               bool
	cacheNewTxEnabled       func() bool
	mu                      sync.RWMutex
	NewLatestSequencerCh     chan bool
}

func (m *IncrementalSyncer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"acquireTxQueue":          len(m.acquireTxQueue),
		"bufferedIncomingTxCache": m.bufferedIncomingTxCache.Len(),
	}
}

type SyncerConfig struct {
	AcquireTxQueueSize                       uint
	MaxBatchSize                             int
	BatchTimeoutMilliSecond                  uint
	AcquireTxDedupCacheMaxSize               int
	AcquireTxDedupCacheExpirationSeconds     int
	BufferedIncomingTxCacheEnabled           bool
	BufferedIncomingTxCacheMaxSize           int
	BufferedIncomingTxCacheExpirationSeconds int
	FiredTxCacheMaxSize                      int
	FiredTxCacheExpirationSeconds            int
}

func NewIncrementalSyncer(config *SyncerConfig, messageSender MessageSender, getTxsHashes func() []types.Hash,
	isKnownHash func(hash types.Hash) bool, getHeight func() uint64, cacheNewTxEnabled func() bool) *IncrementalSyncer {
	return &IncrementalSyncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		bufferedIncomingTxCache: txcache.NewTxCache(config.BufferedIncomingTxCacheMaxSize,
			config.BufferedIncomingTxCacheExpirationSeconds, isKnownHash),
		firedTxCache: gcache.New(config.BufferedIncomingTxCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.BufferedIncomingTxCacheExpirationSeconds)).Build(),
		quitLoopSync:      make(chan bool),
		quitLoopEvent:     make(chan bool),
		EnableEvent:       make(chan bool),
		notifyTxEvent:     make(chan bool),
		NewLatestSequencerCh: make(chan  bool),
		quitNotifyEvent :make (chan bool),
		Enabled:           false,
		getTxsHashes:      getTxsHashes,
		isKnownHash:       isKnownHash,
		getHeight:         getHeight,
		cacheNewTxEnabled: cacheNewTxEnabled,
	}
}

func (m *IncrementalSyncer) Start() {
	go m.eventLoop()
	go m.loopSync()
	go m.txNotifyLoop()
}

func (m *IncrementalSyncer) Stop() {
	m.Enabled = false
	m.quitLoopEvent <- true
	m.quitLoopSync <- true
	m.quitLoopEvent <-true
	// <-ffchan.NewTimeoutSender(m.quitLoopEvent, true, "increSyncerQuitLoopEvent", 1000).C
	// <-ffchan.NewTimeoutSender(m.quitLoopSync, true, "increSyncerQuitLoopSync", 1000).C
}

func (m *IncrementalSyncer) Name() string {
	return "IncrementalSyncer"
}

func (m *IncrementalSyncer) fireRequest(buffer map[types.Hash]struct{}) {
	if len(buffer) == 0 {
		return
	}
	req := types.MessageSyncRequest{
		RequestId: og.MsgCounter.Get(),
	}
	for key := range buffer {
		// add it to the missing queue in case no one responds us.
		// will retry after some time
		if history, err := m.firedTxCache.GetIFPresent(key); err != nil {
			m.firedTxCache.Set(key, FireHistory{
				FiredTimes: 1,
				StartTime:  time.Now(),
				LastTime:   time.Now(),
			})
		} else {
			h := history.(FireHistory)
			h.FiredTimes++
			h.LastTime = time.Now()
			m.firedTxCache.Set(key, h)
		}
		req.Hashes = append(req.Hashes, key)
		//req.Hashes = append(req.Hashes, key)
	}
	log.WithField("type", og.MessageTypeFetchByHashRequest).
		WithField("length", len(req.Hashes)).WithField("hashes", req.String()).Debugf(
		"sending message MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(og.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	//todo optimize later
	//get source msg
	soucrHash := req.Hashes[0]

	m.messageSender.MulticastToSource(og.MessageTypeFetchByHashRequest, &req, &soucrHash)
}

// LoopSync checks if there is new hash to fetcs. Dedup.
func (m *IncrementalSyncer) loopSync() {
	buffer := make(map[types.Hash]struct{})
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	pauseCheckDuration := time.Duration(time.Second)
	var fired int
	for {
		//if paused wait until resume
		if !m.Enabled {
			select {
			case <-m.quitLoopSync:
				log.Info("syncer received quit message. Quitting...")
				return
			case <-time.After(pauseCheckDuration):
				continue
			}
		}
		select {
		case <-m.quitLoopSync:
			log.Info("syncer received quit message. Quitting...")
			return
		case hash := <-m.acquireTxQueue:
			// collect to the set so that we can query in batch
			buffer[hash] = struct{}{}
			if len(buffer) >= m.config.MaxBatchSize {
				fired++
				//bloom filter msg is large , don't send too frequently
				if fired%BloomFilterRate == 0 {
					m.sendBloomFilter(buffer)
					fired = 0
				} else {
					m.fireRequest(buffer)
					buffer = make(map[types.Hash]struct{})
				}
			}
		case <-time.After(sleepDuration):
			// trigger the message if we do not have new queries in such duration
			// check duplicate here in the future
			fired++
			//bloom filter msg is large , don't send too frequently
			if fired%BloomFilterRate == 0 {
				m.sendBloomFilter(buffer)
				fired = 0
			} else {
				m.fireRequest(buffer)
				buffer = make(map[types.Hash]struct{})
			}
		case <-time.After(time.Second * 5):
			repickedHashes := m.repickHashes()
			log.WithField("hashes", repickedHashes).Info("syncer repicked hashes")
			for _, hash := range repickedHashes {
				buffer[hash] = struct{}{}
			}
		}
	}
}

func (m *IncrementalSyncer) Enqueue(hash types.Hash) {
	if !m.Enabled {
		log.WithField("hash", hash).Info("sync task is ignored since syncer is paused")
		return
	}
	if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
		log.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	if txi := m.bufferedIncomingTxCache.Get(hash); txi != nil {
		m.bufferedIncomingTxCache.MoveFront(txi)
		log.WithField("hash", hash).Debugf("already in the bufferedCache. Will be announced later")
		return
	}
	m.acquireTxDedupCache.Set(hash, struct{}{})

	m.acquireTxQueue <- hash
	// <-ffchan.NewTimeoutSender(m.acquireTxQueue, hash, "timeoutAcquireTx", 1000).C
}

func (m *IncrementalSyncer) ClearQueue() {
	// clear all pending tasks
	for len(m.acquireTxQueue) > 0 {
		<-m.acquireTxQueue
	}
	m.acquireTxDedupCache.Purge()
}

func (m *IncrementalSyncer) eventLoop() {
	for {
		select {
		case v := <-m.EnableEvent:
			log.WithField("enable", v).Info("incremental syncer got enable event ")
			//old := m.Enabled
			m.Enabled = v
			m.notifyTxEvent <-true
			//if !old && v {
		// changed from disable to enable.
		//go m.notifyAllCachedTxs()
		//}
		//notify txs from cached first and enable to receive new tx from p2p

		case <-m.quitLoopEvent:
			m.Enabled = false
			log.Info("incremental syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}

func (m*IncrementalSyncer)txNotifyLoop() {
	for {
		select {
		case  <-time.After(20*time.Microsecond):
			   go m.notifyNewTxi()
		case <-m.notifyTxEvent:
			  go m.notifyNewTxi()
         case <-m.NewLatestSequencerCh:
         	log.Debug("sequencer updated")
         	go m.RemoveConfirmedFromCache()
		case <-m.quitNotifyEvent:
			m.Enabled = false
			log.Info("incremental syncer txNotifyLoop received quit message. Quitting...")
			return
		}
	}
}




func (m *IncrementalSyncer) sendBloomFilter(buffer map[types.Hash]struct{}) {
	if len(buffer) == 0 {
		return
	}
	req := types.MessageSyncRequest{
		Filter:    types.NewDefaultBloomFilter(),
		RequestId: og.MsgCounter.Get(),
	}
	height := m.getHeight()
	hashs := m.getTxsHashes()
	for _, hash := range hashs {
		req.Filter.AddItem(hash.Bytes[:])
		req.Height = &height
	}
	err := req.Filter.Encode()
	if err != nil {
		log.WithError(err).Warn("encode filter err")
	}
	log.WithField("type", og.MessageTypeFetchByHashRequest).
		WithField("hash length", len(hashs)).WithField("filter length", len(req.Filter.Data)).Debugf(
		"sending bloom filter  MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(og.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	if len(hashs) == 0 {
		//source unknown
		m.messageSender.MulticastMessage(og.MessageTypeFetchByHashRequest, &req)
		return
	}
	var sourceHash *types.Hash
	for k := range buffer {
		sourceHash = &k
		break
	}
	m.messageSender.MulticastToSource(og.MessageTypeFetchByHashRequest, &req, sourceHash)
}

func (m *IncrementalSyncer) HandleNewTx(newTx *types.MessageNewTx) {
	tx := newTx.RawTx.Tx()
	if tx == nil {
		log.Debug("empty MessageNewTx")
		return
	}

	// cancel pending requests if it is there
	m.firedTxCache.Remove(tx.Hash)
	//notify channel will be  blocked if tps is high ,check first and add
	start := time.Now()
	m.AddTxs(nil,nil,tx)
	end :=time.Now()
	log.WithField("used time for add",end.Sub(start).String()).WithField("q", newTx).Debug("incremental received MessageNewTx")

}

func (m *IncrementalSyncer) HandleNewTxs(newTxs *types.MessageNewTxs) {
	txs := newTxs.RawTxs.ToTxs()
	if txs == nil {
		log.Debug("Empty MessageNewTx")
		return
	}

	for _, tx := range txs {
		m.firedTxCache.Remove(tx.Hash)
	}
	log.WithField("q", newTxs).Debug("incremental received MessageNewTxs")
	m.AddTxs(txs,nil,nil)
}

func (m *IncrementalSyncer) HandleNewSequencer(newSeq *types.MessageNewSequencer) {
	seq := newSeq.RawSequencer.Sequencer()
	if seq == nil {
		log.Debug("empty NewSequence")
		return
	}
	m.firedTxCache.Remove(seq.Hash)
	log.WithField("q", newSeq).Debug("incremental received NewSequence")
	m.AddTxs(nil,nil,seq)
}

func (m *IncrementalSyncer) notifyNewTxi() {
	if !m.Enabled || m.GetNotifying() {
		return
	}
	m.SetNotifying(true)
	defer  m.SetNotifying(false)
	for m.bufferedIncomingTxCache.Len() != 0 {
		if !m.Enabled {
			return
		}
		txi := m.bufferedIncomingTxCache.DeQueue()
		if txi != nil && !m.isKnownHash(txi.GetTxHash()) {
			for _, c := range m.OnNewTxiReceived {
				c <- txi
				// <-ffchan.NewTimeoutSenderShort(c, txi, fmt.Sprintf("syncerNotifyNewTxi_%d", i)).C
			}
		}
	}
}


/*
func (m *IncrementalSyncer) notifyAllCachedTxs() {
	log.WithField("size", m.bufferedIncomingTxCache.Len()).Debug("incoming cache is being dumped")
	txs := m.bufferedIncomingTxCache.PopALl()
	for _, tx := range txs {
		// announce and then remove
		if m.isKnownHash(tx.GetTxHash()) {
			log.WithField("tx ", tx).Debug("duplicated tx ")
			continue
		}
		m.notifyNewTxi(tx)
	}
	log.WithField("size", len(txs)).Debug("incoming cache dumped")
}
*/
func (m *IncrementalSyncer) HandleFetchByHashResponse(syncResponse *types.MessageSyncResponse, sourceId string) {
	if (syncResponse.RawTxs == nil || len(syncResponse.RawTxs) == 0) &&
		(syncResponse.RawSequencers == nil || len(syncResponse.RawSequencers) == 0) {
		log.Debug("empty MessageSyncResponse")
		return
	}

	for _, rawTx := range syncResponse.RawTxs {
		tx := rawTx.Tx()
		if tx ==nil {
			continue
		}
		log.WithField("tx", tx).WithField("peer", sourceId).Debug("received sync response Tx")
		m.firedTxCache.Remove(tx.Hash)
		//m.bufferedIncomingTxCache.Remove(tx.Hash)
		m.AddTxs(nil,nil,tx	)
	}
	for _, v := range syncResponse.RawSequencers {
		seq := v.Sequencer()
		if seq ==nil {
			continue
		}
		log.WithField("seq", seq).WithField("peer", sourceId).Debug("received sync response seq")
		m.firedTxCache.Remove(v.Hash)
		m.AddTxs(nil,nil,seq	)
	}
}

func (m *IncrementalSyncer) repickHashes() types.Hashes {
	maps := m.firedTxCache.GetALL()
	duration := time.Duration(time.Second * 10)
	var result []types.Hash
	for ik, iv := range maps {
		v := iv.(FireHistory)
		if time.Now().Sub(v.LastTime) > duration {
			// haven't got response after 10 seconds
			result = append(result, ik.(types.Hash))
		}
	}
	return result
}

func (m*IncrementalSyncer)addTx(tx types.Txi) error {
	if m.isKnownHash(tx.GetTxHash()) {
		log.WithField("tx ", tx).Debug("duplicated tx received")
		return nil
	}
	if !m.Enabled {
		log.WithField("tx ", tx).Debug("cache txs for future.")
	}
	return  m.bufferedIncomingTxCache.EnQueue(tx)

}


func (m *IncrementalSyncer) AddTxs(txs types.Txs,seqs types.Sequencers, txi types.Txi) error {
	if !m.Enabled && !m.cacheNewTxEnabled() {
		log.Debug("incremental received nexTx but sync disabled")
		return nil
	}
	var err error
	if len(txs)!=0 {
		for _, tx := range txs {
			err = m.addTx(tx)
			if err !=nil  {
				goto out
			}
		}
	}else if len(seqs)!=0 {
		for _, tx := range seqs {
			err = m.addTx(tx)
			if err !=nil{
				goto out
			}
		}
	}else {
		err =  m.addTx(txi)
	}

	out :
	if err!=nil {
		log.WithError(err).Warn("add tx err")
	}
	//m.notifyTxEvent <- true
    return err
}

func (m *IncrementalSyncer) SetNotifying(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifying = v
}

func (m *IncrementalSyncer) GetNotifying() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifying
}

func ( m*IncrementalSyncer)RemoveConfirmedFromCache () {
	log.WithField("total cache item ",  m.bufferedIncomingTxCache.Len()).Info("removing expired item")
	m.bufferedIncomingTxCache.RemoveExpiredAndInvalid(100)
	log.WithField("total cache item ",  m.bufferedIncomingTxCache.Len()).Info("removed expired item")
}
