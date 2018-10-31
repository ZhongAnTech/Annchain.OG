package syncer

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"time"
	"github.com/annchain/OG/ffchan"
)

type MessageSender interface {
	BroadcastMessage(messageType og.MessageType, message []byte)
	UnicastMessageRandomly(messageType og.MessageType, message []byte)
}

// IncrementalSyncer fetches tx from other  peers. (incremental)
// IncrementalSyncer will not fire duplicate requests in a period of time.
type IncrementalSyncer struct {
	config              *SyncerConfig
	messageSender       MessageSender
	acquireTxQueue      chan types.Hash
	acquireTxDedupCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	quitLoopSync        chan bool
	quitLoopEvent       chan bool
	EnableEvent         chan bool
	Enabled             bool
	OnNewTxiReceived    []chan types.Txi
}

func (m *IncrementalSyncer) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{
		"acquireTxQueue": len(m.acquireTxQueue),
	}
}

type SyncerConfig struct {
	AcquireTxQueueSize                   uint
	MaxBatchSize                         int
	BatchTimeoutMilliSecond              uint
	AcquireTxDedupCacheMaxSize           int
	AcquireTxDedupCacheExpirationSeconds int
}

func NewIncrementalSyncer(config *SyncerConfig, messageSender MessageSender) *IncrementalSyncer {
	return &IncrementalSyncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		quitLoopSync:     make(chan bool),
		quitLoopEvent:    make(chan bool),
		EnableEvent:      make(chan bool),
		Enabled:          false,
	}
}

func DefaultSyncerConfig() SyncerConfig {
	config := SyncerConfig{
		BatchTimeoutMilliSecond:              1000,
		AcquireTxQueueSize:                   1000,
		MaxBatchSize:                         100,
		AcquireTxDedupCacheMaxSize:           10000,
		AcquireTxDedupCacheExpirationSeconds: 60,
	}
	return config
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
		Hashes: []types.Hash{},
	}
	for key := range buffer {
		req.Hashes = append(req.Hashes, key)
	}
	bytes, err := req.MarshalMsg(nil)
	if err != nil {
		logrus.WithError(err).Warnf("failed to marshal request: %+v", req)
		return
	}
	logrus.WithField("type", og.MessageTypeFetchByHashRequest).
		WithField("length", len(req.Hashes)).
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
				logrus.Info("syncer received quit message. Quitting...")
				return
			case <-time.After(pauseCheckDuration):
				continue
			}
		}
		select {
		case <-m.quitLoopSync:
			logrus.Info("syncer received quit message. Quitting...")
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
		}
	}
}

func (m *IncrementalSyncer) Enqueue(hash types.Hash) {
	if !m.Enabled {
		logrus.WithField("hash", hash).Info("sync task is ignored since syncer is paused")
		return
	}
	if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
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
			m.Enabled = v
		case <-m.quitLoopEvent:
			logrus.Info("incremental syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}

func (s *IncrementalSyncer) HandleNewTx(newTx types.MessageNewTx) {
	if !s.Enabled {
		logrus.Debug("incremental received tx but sync disabled")
		return
	}

	logrus.WithField("q", newTx).Debug("incremental received MessageNewTx")
	if newTx.Tx == nil {
		logrus.Debug("empty MessageNewTx")
		return
	}

	s.notifyNewTxi(newTx.Tx)
}

func (s *IncrementalSyncer) HandleNewTxs(newTxs types.MessageNewTxs) {
	if !s.Enabled {
		logrus.Debug("incremental received txs but sync disabled")
		return
	}
	logrus.WithField("q", newTxs).Debug("incremental received MessageNewTxs")
	//if s.SyncManager.Status != syncer.SyncStatusIncremental{
	//	return
	//}
	if newTxs.Txs == nil {
		logrus.Debug("Empty MessageNewTx")
		return
	}

	for _, tx := range newTxs.Txs {
		s.notifyNewTxi(tx)
	}
}

func (s *IncrementalSyncer) HandleNewSequencer(newSeq types.MessageNewSequencer) {
	if !s.Enabled {
		logrus.Debug("incremental received seq but sync disabled")
		return
	}
	logrus.WithField("q", newSeq).Debug("incremental received NewSequence")
	//if s.SyncManager.Status != syncer.SyncStatusIncremental{
	//	return
	//}
	if newSeq.Sequencer == nil {
		logrus.Debug("empty NewSequence")
		return
	}
	s.notifyNewTxi(newSeq.Sequencer)
}

func (s *IncrementalSyncer) notifyNewTxi(txi types.Txi) {
	for _, c := range s.OnNewTxiReceived {
		c <- txi
	}
}
