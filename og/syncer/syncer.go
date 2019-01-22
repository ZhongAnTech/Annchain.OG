package syncer

import (
	"github.com/annchain/OG/og/txcache"
	"sync"
	"time"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
)

const BloomFilterRate = 5 //sending 4 req

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
	acquireTxQueue          chan *types.Hash
	acquireTxDedupCache     gcache.Cache     // list of hashes that are queried recently. Prevent duplicate requests.
	bufferedIncomingTxCache *txcache.TxCache // cache of incoming txs that are not fired during full sync.
	firedTxCache            gcache.Cache     // cache of hashes that are fired however haven't got any response yet
	quitLoopSync            chan bool
	quitLoopEvent           chan bool
	quitNotifyEvent         chan bool
	EnableEvent             chan bool
	Enabled                 bool
	OnNewTxiReceived        []chan []types.Txi
	notifyTxEvent           chan bool
	notifying               bool
	cacheNewTxEnabled       func() bool
	mu                      sync.RWMutex
	NewLatestSequencerCh    chan bool
	bloomFilterStatus       *BloomFilterFireStatus
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
	NewTxsChannelSize                        int
}

func NewIncrementalSyncer(config *SyncerConfig, messageSender MessageSender, getTxsHashes func() []types.Hash,
	isKnownHash func(hash types.Hash) bool, getHeight func() uint64, cacheNewTxEnabled func() bool) *IncrementalSyncer {
	return &IncrementalSyncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan *types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		bufferedIncomingTxCache: txcache.NewTxCache(config.BufferedIncomingTxCacheMaxSize,
			config.BufferedIncomingTxCacheExpirationSeconds, isKnownHash, true),
		firedTxCache: gcache.New(config.BufferedIncomingTxCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.BufferedIncomingTxCacheExpirationSeconds)).Build(),
		quitLoopSync:         make(chan bool),
		quitLoopEvent:        make(chan bool),
		EnableEvent:          make(chan bool),
		notifyTxEvent:        make(chan bool),
		NewLatestSequencerCh: make(chan bool),
		quitNotifyEvent:      make(chan bool),
		Enabled:              false,
		getTxsHashes:         getTxsHashes,
		isKnownHash:          isKnownHash,
		getHeight:            getHeight,
		cacheNewTxEnabled:    cacheNewTxEnabled,
		bloomFilterStatus:NewBloomFilterFireStatus(120,500),
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
	m.quitLoopEvent <- true
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
	var triggerTime int
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	pauseCheckDuration := time.Duration(time.Millisecond*100)
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
		case v := <-m.acquireTxQueue:
			// collect to the set so that we can query in batch
			if v!=nil {
				hash:= *v
				buffer[hash] = struct{}{}
			}
			triggerTime++
			if len(buffer) >0 && triggerTime >= m.config.MaxBatchSize  {
				//bloom filter msg is large , don't send too frequently
				if fired%BloomFilterRate == 0 {
					var hash types.Hash
					for hash = range buffer {
						break
					}
					m.sendBloomFilter(hash)
					fired = 0
				} else {
					m.fireRequest(buffer)
					buffer = make(map[types.Hash]struct{})
					triggerTime = 0
				}
				fired++
			}
		case <-time.After(sleepDuration):
			// trigger the message if we do not have new queries in such duration
			// check duplicate here in the future
			//bloom filter msg is large , don't send too frequently
			if len(buffer)>0 {
				if fired%BloomFilterRate == 0 {
					var hash types.Hash
					for hash = range buffer {
						break
					}
					m.sendBloomFilter(hash)
					fired = 0
				} else {
					m.fireRequest(buffer)
					buffer = make(map[types.Hash]struct{})
					triggerTime = 0
				}
				fired++
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

func (m *IncrementalSyncer) Enqueue(phash *types.Hash, sendBloomfilter bool) {
	if !m.Enabled {
		log.WithField("hash", phash).Info("sync task is ignored since syncer is paused")
		return
	}
	if phash!=nil {
		hash:= *phash
		if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
			log.WithField("hash", hash).Debugf("duplicate sync task")
			return
		}
		if  m.bufferedIncomingTxCache.Has(hash) {
			log.WithField("hash", hash).Debugf("already in the bufferedCache. Will be announced later")
			return
		}
		m.acquireTxDedupCache.Set(hash, struct{}{})
		if sendBloomfilter {
			go m.sendBloomFilter(hash)
		}
	}
	m.acquireTxQueue <- phash
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
			m.notifyTxEvent <- true
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

func (m *IncrementalSyncer) txNotifyLoop() {
	for {
		select {
		case <-time.After(20 * time.Microsecond):
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


func (m *IncrementalSyncer) notifyNewTxi() {
	if !m.Enabled || m.GetNotifying() {
		return
	}
	m.SetNotifying(true)
	defer m.SetNotifying(false)
	for m.bufferedIncomingTxCache.Len() != 0 {
		if !m.Enabled {
			break
		}
		log.Trace("len cache ", m.bufferedIncomingTxCache.Len())
		txis, err := m.bufferedIncomingTxCache.DeQueueBatch(m.config.NewTxsChannelSize)
		if err != nil {
			log.WithError(err).Warn("got tx failed")
			break
		}
		log.Trace("got txis ", txis)
		var txs types.Txis
		for _, txi := range txis {
			if txi != nil && !m.isKnownHash(txi.GetTxHash()) {
				txs = append(txs, txi)
			}
		}
		for _, c := range m.OnNewTxiReceived {
			c <- txs
			// <-ffchan.NewTimeoutSenderShort(c, txi, fmt.Sprintf("syncerNotifyNewTxi_%d", i)).C
		}
		log.Trace("len cache ", m.bufferedIncomingTxCache.Len())
	}
	if m.bufferedIncomingTxCache.Len() != 0 {
		log.Debug("len cache ", m.bufferedIncomingTxCache.Len())
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

func (m *IncrementalSyncer) RemoveConfirmedFromCache() {
	log.WithField("total cache item ", m.bufferedIncomingTxCache.Len()).Info("removing expired item")
	m.bufferedIncomingTxCache.RemoveExpiredAndInvalid(60)
	log.WithField("total cache item ", m.bufferedIncomingTxCache.Len()).Info("removed expired item")
}
