package og

import (
	"github.com/annchain/OG/types"
	"time"
	"github.com/sirupsen/logrus"
)

// Syncer fetches tx from other peers.
type Syncer struct {
	config         *SyncerConfig
	hub            *Hub
	acquireTxQueue chan types.Hash
	quit           chan bool
}

type SyncerConfig struct {
	AcquireTxQueueSize      uint
	MaxBatchSize            int
	BatchTimeoutMilliSecond uint
}

func NewSyncer(config *SyncerConfig, hub *Hub) *Syncer {
	return &Syncer{
		config:         config,
		hub:            hub,
		acquireTxQueue: make(chan types.Hash, config.AcquireTxQueueSize),
		quit:           make(chan bool),
	}
}

func (m *Syncer) Start() {
	go m.loopSync()
}

func (m *Syncer) Stop() {
	m.quit <- true
}

func (m *Syncer) Name() string {
	return "Manager"
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
		logrus.WithError(err).Warnf("Failed to marshal request: %+v", req)
		return
	}
	logrus.WithField("type", MessageTypeFetchByHash).
		WithField("length", len(req.Hashes)).
		Debugf("Sending message MessageTypeFetchByHash")

	m.hub.outgoing <- &P2PMessage{
		MessageType: MessageTypeFetchByHash,
		Message:     bytes,
	}
}

// LoopSync checks if there is new hash to fetch. Dedup.
func (m *Syncer) loopSync() {
	buffer := make(map[types.Hash]struct{})
	sleepDuration := time.Duration(m.config.BatchTimeoutMilliSecond) * time.Millisecond
	for {
		select {
		case <-m.quit:
			logrus.Info("Syncer reeived quit message. Quitting...")
			return
		case hash := <-m.acquireTxQueue:
			// collect to the set so that we can query in batch
			buffer[hash] = struct{}{}
			if len(buffer) >= m.config.MaxBatchSize{
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
	m.acquireTxQueue <- hash
}
