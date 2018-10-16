package og

import (
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"time"
)

// Syncer fetches tx from other peers. (incremental)
// Syncer will not fire duplicate requests in a period of time.
type Syncer struct {
	config              *SyncerConfig
	hub                 *Hub
	acquireTxQueue      chan types.Hash
	acquireTxDedupCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	quit                chan bool
	pause               chan bool
	EnableEvent         chan bool
	paused              bool
}

func (m *Syncer) GetBenchmarks() map[string]int {
	return map[string]int{
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

func NewSyncer(config *SyncerConfig, hub *Hub) *Syncer {
	return &Syncer{
		config:         config,
		hub:            hub,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		acquireTxDedupCache: gcache.New(config.AcquireTxDedupCacheMaxSize).Simple().
			Expiration(time.Second * time.Duration(config.AcquireTxDedupCacheExpirationSeconds)).Build(),
		quit:        make(chan bool),
		pause:       make(chan bool),
		EnableEvent: make(chan bool),
		paused:      true,
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
	m.quit <- true
}

func (m *Syncer) Pause() {
	m.pause <- true
}

func (m *Syncer) Resume() {
	m.pause <- false
	m.paused = false
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

	m.hub.SendMessage(MessageTypeFetchByHash, bytes)
}

// LoopSync checks if there is new hash to fetch. Dedup.
func (m *Syncer) loopSync() {
	buffer := make(map[types.Hash]struct{})
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond

	for {
		//if paused wait until resume
		if m.paused {
			select {
			case v := <-m.pause:
				m.paused = v
				logrus.WithField("value", v).Debug("syncer received pause/resume message.")
			case <-m.quit:
				logrus.Info("syncer received quit message. Quitting...")
				return
			}
		}
		select {
		case v := <-m.pause:
			m.paused = v
			logrus.WithField("value", v).Debug("syncer received pause/resume message.")
		case <-m.quit:
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
	if m.paused {
		logrus.WithField("hash", hash).Info("sync task is ignored since syncer is paused")
		return
	}
	if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	m.acquireTxDedupCache.Set(hash, struct{}{})
	m.acquireTxQueue <- hash
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
			logrus.WithField("enable: ", v).Info("got enable event ")
			if v {
				m.Resume()
			} else {
				m.Pause()
			}
		case <-m.quit:
			logrus.Info("syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}
