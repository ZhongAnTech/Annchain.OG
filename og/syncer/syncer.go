package syncer

import (
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"time"
	"fmt"
)

type MessageSender interface {
	BroadcastMessage(messageType og.MessageType, message []byte)
	UnicastMessageRandomly(messageType og.MessageType, message []byte)
}

type FireHistory struct {
	StartTime  time.Time
	LastTime   time.Time
	FiredTimes int
}

// IncrementalSyncer fetches tx from other  peers. (incremental)
// IncrementalSyncer will not fire duplicate requests in a period of time.
type IncrementalSyncer struct {
	config                         *SyncerConfig
	messageSender                  MessageSender
	acquireTxQueue                 chan types.Hash
	acquireTxDedupCache            gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	bufferedIncomingTxCache        gcache.Cache // cache of incoming txs that are not fired during full sync.
	firedTxCache                   gcache.Cache // cache of hashes that are fired however haven't got any response yet
	bufferedIncomingTxCacheEnabled bool
	quitLoopSync                   chan bool
	quitLoopEvent                  chan bool
	EnableEvent                    chan bool
	Enabled                        bool
	OnNewTxiReceived               []chan types.Txi
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

func NewIncrementalSyncer(config *SyncerConfig, messageSender MessageSender) *IncrementalSyncer {
	return &IncrementalSyncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		bufferedIncomingTxCache: gcache.New(config.BufferedIncomingTxCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.BufferedIncomingTxCacheExpirationSeconds)).Build(),
		firedTxCache: gcache.New(config.BufferedIncomingTxCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.BufferedIncomingTxCacheExpirationSeconds)).Build(),
		bufferedIncomingTxCacheEnabled: config.BufferedIncomingTxCacheEnabled,
		quitLoopSync:                   make(chan bool),
		quitLoopEvent:                  make(chan bool),
		EnableEvent:                    make(chan bool),
		Enabled:                        false,
	}
}

func (m *IncrementalSyncer) Start() {
	go m.eventLoop()
	go m.loopSync()
}

func (m *IncrementalSyncer) Stop() {
	m.Enabled = false
	<-ffchan.NewTimeoutSender(m.quitLoopEvent, true, "increSyncerQuitLoopEvent", 1000).C
	<-ffchan.NewTimeoutSender(m.quitLoopSync, true, "increSyncerQuitLoopSync", 1000).C
	//m.quitLoopEvent <- true
	//m.quitLoopSync <- true
}

func (m *IncrementalSyncer) Name() string {
	return "IncrementalSyncer"
}

func (m *IncrementalSyncer) fireRequest(buffer map[types.Hash]struct{}) {
	if len(buffer) == 0 {
		return
	}
	req := types.MessageSyncRequest{
		Hashes:    []types.Hash{},
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
	}
	bytes, err := req.MarshalMsg(nil)
	if err != nil {
		log.WithError(err).Warnf("failed to marshal request: %+v", req)
		return
	}
	log.WithField("type", og.MessageTypeFetchByHashRequest).
		WithField("length", len(req.Hashes)).
		WithField("hashes", types.HashesToString(req.Hashes)).
		Debugf("sending message MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(og.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so brodacst it
	//todo optimize later
	m.messageSender.BroadcastMessage(og.MessageTypeFetchByHashRequest, bytes)
}

// LoopSync checks if there is new hash to fetcs. Dedup.
func (m *IncrementalSyncer) loopSync() {
	buffer := make(map[types.Hash]struct{})
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	pauseCheckDuration := time.Duration(time.Second)

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
				m.fireRequest(buffer)
				buffer = make(map[types.Hash]struct{})
			}
		case <-time.After(sleepDuration):
			// trigger the message if we do not have new queries in such duration
			// check duplicate here in the future
			m.fireRequest(buffer)
			buffer = make(map[types.Hash]struct{})
		case <-time.After(time.Second * 5):
			repickedHashes := m.repickHashes()
			logrus.WithField("hashes", types.HashesToString(repickedHashes)).Info("syncer repicked hashes")
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
	if _, err := m.bufferedIncomingTxCache.GetIFPresent(hash); err == nil {
		log.WithField("hash", hash).Debugf("already in the bufferedCache. Will be announced later")
		return
	}
	m.acquireTxDedupCache.Set(hash, struct{}{})

	<-ffchan.NewTimeoutSender(m.acquireTxQueue, hash, "timeoutAcquireTx", 1000).C
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
			logrus.WithField("enable", v).Info("incremental syncer got enable event ")
			old := m.Enabled
			m.Enabled = v
			if !old && v {
				// changed from disable to enable.
				m.notifyAllCachedTxs()
			}

		case <-m.quitLoopEvent:
			log.Info("incremental syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}

func (m *IncrementalSyncer) HandleNewTx(newTx types.MessageNewTx) {
	if newTx.Tx == nil {
		logrus.Debug("empty MessageNewTx")
		return
	}

	// cancel pending requests if it is there
	m.firedTxCache.Remove(newTx.Tx.Hash)

	if !m.Enabled {
		if !m.bufferedIncomingTxCacheEnabled {
			logrus.Debug("incremental received tx but sync disabled")
			return
		}
		logrus.WithField("tx", newTx.Tx).Trace("cache tx for future.")
		m.bufferedIncomingTxCache.Set(newTx.Tx.Hash, newTx.Tx)
		return
	}

	logrus.WithField("q", newTx).Debug("incremental received MessageNewTx")
	m.notifyNewTxi(newTx.Tx)
}

func (m *IncrementalSyncer) HandleNewTxs(newTxs types.MessageNewTxs) {
	if newTxs.Txs == nil {
		logrus.Debug("Empty MessageNewTx")
		return
	}

	for _, tx := range newTxs.Txs {
		m.firedTxCache.Remove(tx.Hash)
	}

	if !m.Enabled {
		if !m.bufferedIncomingTxCacheEnabled {
			logrus.Debug("incremental received txs but sync disabled")
			return
		}
		for _, tx := range newTxs.Txs {
			logrus.WithField("tx", tx).Trace("cache tx for future.")
			m.bufferedIncomingTxCache.Set(tx.Hash, tx)
		}
		return
	}
	logrus.WithField("q", newTxs).Debug("incremental received MessageNewTxs")

	for _, tx := range newTxs.Txs {
		m.notifyNewTxi(tx)
	}
}

func (m *IncrementalSyncer) HandleNewSequencer(newSeq types.MessageNewSequencer) {
	if newSeq.Sequencer == nil {
		logrus.Debug("empty NewSequence")
		return
	}
	m.firedTxCache.Remove(newSeq.Sequencer.Hash)

	if !m.Enabled {
		if !m.bufferedIncomingTxCacheEnabled {
			logrus.Debug("incremental received seq but sync disabled")
			return
		}
		logrus.WithField("seq", newSeq.Sequencer).Trace("cache seq for future.")
		m.bufferedIncomingTxCache.Set(newSeq.Sequencer.Hash, newSeq.Sequencer)
		return
	}
	logrus.WithField("q", newSeq).Debug("incremental received NewSequence")
	m.notifyNewTxi(newSeq.Sequencer)
}

func (m *IncrementalSyncer) notifyNewTxi(txi types.Txi) {
	for i, c := range m.OnNewTxiReceived {
		<-ffchan.NewTimeoutSenderShort(c, txi, fmt.Sprintf("syncerNotifyNewTxi_%d", i)).C
	}
}

func (m *IncrementalSyncer) notifyAllCachedTxs() {
	logrus.WithField("size", m.bufferedIncomingTxCache.Len()).Trace("incoming cache is being dumped")
	kvMap := m.bufferedIncomingTxCache.GetALL()
	for k, v := range kvMap {
		// annouce and then remove
		m.notifyNewTxi(v.(types.Txi))
		m.bufferedIncomingTxCache.Remove(k)
	}
	logrus.WithField("size", m.bufferedIncomingTxCache.Len()).Trace("incoming cache dumped")
}

func (m *IncrementalSyncer) HandleFetchByHashResponse(syncResponse types.MessageSyncResponse, sourceId string) {
	if (syncResponse.Txs == nil || len(syncResponse.Txs) == 0) &&
		(syncResponse.Sequencers == nil || len(syncResponse.Sequencers) == 0) {
		logrus.Debug("empty MessageSyncResponse")
		return
	}

	for _, v := range syncResponse.Txs {
		logrus.WithField("tx", v).WithField("peer", sourceId).Debug("received sync response Tx")
		m.firedTxCache.Remove(v.Hash)
		m.notifyNewTxi(v)
	}
	for _, v := range syncResponse.Sequencers {
		logrus.WithField("seq", v).WithField("peer", sourceId).Debug("received sync response seq")
		m.firedTxCache.Remove(v.Hash)
		m.notifyNewTxi(v)
	}
}
func (m *IncrementalSyncer) repickHashes() []types.Hash {
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
