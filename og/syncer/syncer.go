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
package syncer

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"

	"github.com/annchain/OG/types/msg"
	"sync"
	"time"

	"github.com/annchain/OG/og/txcache"

	"github.com/annchain/gcache"
)

const BloomFilterRate = 4 //sending 4 req

type MessageSender interface {
	BroadcastMessage(message msg.OgMessage)
	MulticastMessage(message msg.OgMessage)
	MulticastToSource(message msg.OgMessage, sourceMsgHash *common.Hash)
	BroadcastMessageWithLink(message msg.OgMessage)
	SendToPeer(peerId string, msg msg.OgMessage)
}

type FireHistory struct {
	StartTime  time.Time
	LastTime   time.Time
	FiredTimes int
}

// IncrementalSyncer fetches tx from other  peers. (incremental)
// IncrementalSyncer will not fire duplicate requests in a period of time.
type IncrementalSyncer struct {
	config                   *SyncerConfig
	messageSender            MessageSender
	getTxsHashes             func() common.Hashes
	isKnownHash              func(hash common.Hash) bool
	getHeight                func() uint64
	acquireTxQueue           chan *common.Hash
	acquireTxDuplicateCache  gcache.Cache     // list of hashes that are queried recently. Prevent duplicate requests.
	bufferedIncomingTxCache  *txcache.TxCache // cache of incoming txs that are not fired during full sync.
	firedTxCache             gcache.Cache     // cache of hashes that are fired however haven't got any response yet
	quitLoopSync             chan bool
	quitLoopEvent            chan bool
	quitNotifyEvent          chan bool
	EnableEvent              chan bool
	Enabled                  bool
	OnNewTxiReceived         []chan []types.Txi
	notifyTxEvent            chan bool
	notifying                bool
	cacheNewTxEnabled        func() bool
	mu                       sync.RWMutex
	NewLatestSequencerCh     chan bool
	bloomFilterStatus        *BloomFilterFireStatus
	RemoveContrlMsgFromCache func(hash common.Hash)
	SequencerCache           *SequencerCache
}

func (m *IncrementalSyncer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"acquireTxQueue":          len(m.acquireTxQueue),
		"bufferedIncomingTxCache": m.bufferedIncomingTxCache.Len(),
		"firedTxCache":            m.firedTxCache.Len(true),
		"acquireTxDuplicateCache": m.acquireTxDuplicateCache.Len(true),
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

func NewIncrementalSyncer(config *SyncerConfig, messageSender MessageSender, getTxsHashes func() common.Hashes,
	isKnownHash func(hash common.Hash) bool, getHeight func() uint64, cacheNewTxEnabled func() bool) *IncrementalSyncer {
	return &IncrementalSyncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan *common.Hash, config.AcquireTxQueueSize),
		acquireTxDuplicateCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
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
		bloomFilterStatus:    NewBloomFilterFireStatus(120, 500),
		SequencerCache:       NewSequencerCache(15),
	}
}

func (m *IncrementalSyncer) Start() {
	goroutine.New(m.eventLoop)
	goroutine.New(m.loopSync)
	goroutine.New(m.txNotifyLoop)
}

func (m *IncrementalSyncer) Stop() {
	m.Enabled = false
	close(m.quitLoopEvent)
	close(m.quitLoopSync)
	close(m.quitNotifyEvent)
	// <-ffchan.NewTimeoutSender(m.quitLoopEvent, true, "increSyncerQuitLoopEvent", 1000).C
	// <-ffchan.NewTimeoutSender(m.quitLoopSync, true, "increSyncerQuitLoopSync", 1000).C
}

func (m *IncrementalSyncer) Name() string {
	return "IncrementalSyncer"
}

func (m *IncrementalSyncer) CacheTxs(txs types.Txis) {
	m.bufferedIncomingTxCache.EnQueueBatch(txs)
}

func (m *IncrementalSyncer) CacheTx(tx types.Txi) {
	m.bufferedIncomingTxCache.EnQueue(tx)
}

func (m *IncrementalSyncer) fireRequest(buffer map[common.Hash]struct{}) {
	if len(buffer) == 0 {
		return
	}
	req := archive.MessageSyncRequest{
		RequestId: message_archive.MsgCounter.Get(),
	}
	var source interface{}
	var err error
	var reqHashes common.Hashes
	for key := range buffer {
		if source, err = m.acquireTxDuplicateCache.GetIFPresent(key); err != nil {
			continue
		}
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
		reqHashes = append(reqHashes, key)
		//req.Hashes = append(req.Hashes, key)
	}
	if len(reqHashes) == 0 {
		return
	}
	req.Hashes = &reqHashes

	log.WithField("type", req.GetType()).
		WithField("length", len(reqHashes)).WithField("hashes", req.String()).Debugf(
		"sending message")

	//m.messageSender.UnicastMessageRandomly(p2p_message.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	//todo optimize later
	//get source msg
	soucrHash := source.(common.Hash)

	m.messageSender.MulticastToSource(&req, &soucrHash)
}

// LoopSync checks if there is new hash to fetch. Dedup.
func (m *IncrementalSyncer) loopSync() {
	buffer := make(map[common.Hash]struct{})
	var triggerTime int
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	pauseCheckDuration := time.Duration(time.Millisecond * 100)
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
			if v != nil {
				hash := *v
				buffer[hash] = struct{}{}
			}
			triggerTime++
			if len(buffer) > 0 && triggerTime >= m.config.MaxBatchSize {
				//bloom filter msg is large , don't send too frequently
				if fired%BloomFilterRate == 0 {
					var hash common.Hash
					for key := range buffer {
						hash = key
						source, err := m.acquireTxDuplicateCache.GetIFPresent(key)
						if err != nil {
							continue
						}
						hash = source.(common.Hash)
						break
					}
					m.sendBloomFilter(hash)
					fired = 0
				} else {
					m.fireRequest(buffer)
					buffer = make(map[common.Hash]struct{})
					triggerTime = 0
				}
				fired++
			}
		case <-time.After(sleepDuration):
			// trigger the message if we do not have new queries in such duration
			// check duplicate here in the future
			//bloom filter msg is large , don't send too frequently
			if len(buffer) > 0 {
				if fired%BloomFilterRate == 0 {
					var hash common.Hash
					for key := range buffer {
						hash = key
						source, err := m.acquireTxDuplicateCache.GetIFPresent(key)
						if err != nil {
							continue
						}
						hash = source.(common.Hash)
						break
					}
					m.sendBloomFilter(hash)
					fired = 0
				} else {
					m.fireRequest(buffer)
					buffer = make(map[common.Hash]struct{})
					triggerTime = 0
				}
				fired++
			}

		case <-time.After(sleepDuration * 5):
			repickedHashes := m.repickHashes()
			log.WithField("hashes", repickedHashes).Info("syncer repicked hashes")
			for _, hash := range repickedHashes {
				buffer[hash] = struct{}{}
			}
		}
	}
}

func (m *IncrementalSyncer) Enqueue(phash *common.Hash, childHash common.Hash, sendBloomfilter bool) {
	if !m.Enabled {
		log.WithField("hash", phash).Info("sync task is ignored since syncer is paused")
		return
	}
	if phash != nil {
		hash := *phash
		if _, err := m.acquireTxDuplicateCache.Get(hash); err == nil {
			log.WithField("hash", hash).Debugf("duplicate sync task")
			return
		}
		if m.bufferedIncomingTxCache.Has(hash) {
			log.WithField("hash", hash).Debugf("already in the bufferedCache. Will be announced later")
			return
		}
		m.acquireTxDuplicateCache.Set(hash, childHash)
		if sendBloomfilter {
			goroutine.New(func() {
				m.sendBloomFilter(childHash)
			})
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
	m.acquireTxDuplicateCache.Purge()
}

func (m *IncrementalSyncer) eventLoop() {
	for {
		select {
		case v := <-m.EnableEvent:
			log.WithField("enable", v).Info("incremental syncer got enable event")
			//old := m.Enabled
			m.Enabled = v
			m.notifyTxEvent <- true
			//if !old && v {
		// changed from disable to enable.
		//goroutine.New( m.notifyAllCachedTxs )
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
		//todo 20*microsecond to millisecond , check this later
		case <-time.After(20 * time.Millisecond):
			goroutine.New(m.notifyNewTxi)
		case <-m.notifyTxEvent:
			goroutine.New(m.notifyNewTxi)
		case <-m.NewLatestSequencerCh:
			log.Debug("sequencer updated")
			goroutine.New(m.RemoveConfirmedFromCache)
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
			if txi != nil && !m.isKnownHash(txi.GetHash()) {
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
		if m.isKnownHash(tx.GetHash()) {
			log.WithField("tx ", tx).Debug("duplicated tx ")
			continue
		}
		m.notifyNewTxi(tx)
	}
	log.WithField("size", len(txs)).Debug("incoming cache dumped")
}
*/

func (m *IncrementalSyncer) repickHashes() common.Hashes {
	maps := m.firedTxCache.GetALL(true)
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	duration := time.Duration(sleepDuration * 20)
	var result common.Hashes
	for ik, iv := range maps {
		v := iv.(FireHistory)
		if time.Now().Sub(v.LastTime) > duration {
			// haven't got response after 10 seconds
			result = append(result, ik.(common.Hash))
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
	log.WithField("total cache item ", m.bufferedIncomingTxCache.Len()).Debug("removing expired item")
	m.bufferedIncomingTxCache.RemoveExpiredAndInvalid(60)
	log.WithField("total cache item ", m.bufferedIncomingTxCache.Len()).Debug("removed expired item")
}

func (m *IncrementalSyncer) IsCachedHash(hash common.Hash) bool {
	return m.bufferedIncomingTxCache.Has(hash)
}

func (m *IncrementalSyncer) TxEnable() bool {
	return m.Enabled
}

func (m *IncrementalSyncer) SyncHashList(seqHash common.Hash) {
	peerId := m.SequencerCache.GetPeer(seqHash)
	if peerId == "" {
		log.Warn("nil peer id")
		return
	}
	goroutine.New(func() {
		m.syncHashList(peerId)
	})
}

func (m *IncrementalSyncer) syncHashList(peerId string) {
	req := archive.MessageSyncRequest{
		RequestId: message_archive.MsgCounter.Get(),
	}
	height := m.getHeight()
	req.Height = &height
	hashs := m.getTxsHashes()
	var hashTerminates archive.HashTerminats
	for _, hash := range hashs {
		var hashTerminate archive.HashTerminat
		copy(hashTerminate[:], hash.Bytes[:4])
		hashTerminates = append(hashTerminates, hashTerminate)
	}
	req.HashTerminats = &hashTerminates
	log.WithField("to ", peerId).WithField("hash list num ", len(hashTerminates)).
		WithField("height ", height).
		WithField("type", req.GetType()).
		WithField("len ", req.HashTerminats).Debug("sending hashList MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(p2p_message.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	m.messageSender.SendToPeer(peerId, &req)
	return
}
