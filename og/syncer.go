package og

import (
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"time"
)

type MessageSender interface {
	BroadcastMessage(messageType MessageType, message []byte)
	UnicastMessageRandomly(messageType MessageType, message []byte)
}

// Syncer fetches tx from other  peers. (incremental)
// Syncer will not fire duplicate requests in a period of time.
type Syncer struct {
	config              *SyncerConfig
	messageSender       MessageSender
	acquireTxQueue      chan types.Hash
	acquireTxDedupCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	quitLoopSync        chan bool
	quitLoopEvent       chan bool
	EnableEvent         chan bool
	enabled             bool
	timeoutAcquireTx    *time.Timer
}

func (m *Syncer) GetBenchmarks() map[string]interface{} {
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

func NewSyncer(config *SyncerConfig, messageSender MessageSender) *Syncer {
	return &Syncer{
		config:         config,
		messageSender:  messageSender,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		quitLoopSync:     make(chan bool),
		quitLoopEvent:    make(chan bool),
		EnableEvent:      make(chan bool),
		enabled:          false,
		timeoutAcquireTx: time.NewTimer(time.Second * 10),
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

func (m *Syncer) Start() {
	go m.eventLoop()
	go m.loopSync()
}

func (m *Syncer) Stop() {
	m.enabled = false
	m.quitLoopEvent <- true
	m.quitLoopSync <- true
}

func (m *Syncer) Name() string {
	return "Syncer"
}

func (m *Syncer) fireRequest(buffer map[types.Hash]struct{}) {
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
	logrus.WithField("type", MessageTypeFetchByHash).
		WithField("length", len(req.Hashes)).
		Debugf("sending message MessageTypeFetchByHash")

	m.messageSender.UnicastMessageRandomly(MessageTypeFetchByHash, bytes)
}

// LoopSync checks if there is new hash to fetch. Dedup.
func (m *Syncer) loopSync() {
	buffer := make(map[types.Hash]struct{})
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	pauseCheckDuration := time.Duration(time.Second)

	for {
		//if paused wait until resume
		if !m.enabled {
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

func (m *Syncer) Enqueue(hash types.Hash) {
	if !m.enabled {
		logrus.WithField("hash", hash).Info("sync task is ignored since syncer is paused")
		return
	}
	if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	m.acquireTxDedupCache.Set(hash, struct{}{})

loop:
	for {
		if !m.timeoutAcquireTx.Stop() {
			<-m.timeoutAcquireTx.C
		}
		m.timeoutAcquireTx.Reset(time.Second * 10)
		select {
		case <-m.timeoutAcquireTx.C:
			logrus.WithField("hash", hash.String()).Warn("timeout on channel writing: acquire tx")
		case m.acquireTxQueue <- hash:
			break loop
		}
	}

}

func (m *Syncer) ClearQueue() {
	// clear all pending tasks
	for len(m.acquireTxQueue) > 0 {
		<-m.acquireTxQueue
	}
	m.acquireTxDedupCache.Purge()
}

func (m *Syncer) eventLoop() {
	for {
		select {
		case v := <-m.EnableEvent:
			logrus.WithField("enable", v).Info("syncer got enable event ")
			m.enabled = v
		case <-m.quitLoopEvent:
			logrus.Info("syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}

//BroadcastNewTx brodcast newly created txi message
func (m *Syncer) BroadcastNewTx(txi types.Txi) {
	txType := txi.GetType()
	if txType == types.TxBaseTypeNormal {
		tx := txi.(*types.Tx)
		msgTx := types.MessageNewTx{Tx: tx}
		data, _ := msgTx.MarshalMsg(nil)
		m.messageSender.BroadcastMessage(MessageTypeNewTx, data)
	} else if txType == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		msgTx := types.MessageNewSequencer{seq}
		data, _ := msgTx.MarshalMsg(nil)
		m.messageSender.BroadcastMessage(MessageTypeNewSequencer, data)
	} else {
		logrus.Warn("never come here ,unkown tx type", txType)
	}
}

