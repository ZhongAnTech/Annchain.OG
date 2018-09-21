package og

import (
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/sirupsen/logrus"
	"time"
)

// Syncer fetches tx from other peers.
// Syncer will not fire duplicate requests in a period of time.
type Syncer struct {
	config              *SyncerConfig
	hub                 *Hub
	acquireTxQueue      chan types.Hash
	acquireTxDedupCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	quit                chan bool
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
		quit: make(chan bool),
	}
}

func (m *Syncer) Start() {
	go m.loopSync()
}

func (m *Syncer) Stop() {
	m.quit <- true
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
		select {
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

// LoopSwipe maintains all sync requests. Handle if there is timeout.
func (m *Syncer) loopSwipe() {

}

func (m *Syncer) Enqueue(hash types.Hash) {
	if _, err := m.acquireTxDedupCache.Get(hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	m.acquireTxDedupCache.Set(hash, struct{}{})
	m.acquireTxQueue <- hash
}
