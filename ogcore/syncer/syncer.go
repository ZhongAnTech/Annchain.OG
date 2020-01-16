package syncer

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"time"
)

type SyncerConfig struct {
	//AcquireTxQueueSize uint
	//MaxBatchSize                             int
	//BatchTimeoutMilliSecond              int
	AcquireTxDedupCacheMaxSize           int
	AcquireTxDedupCacheExpirationSeconds int
	//BufferedIncomingTxCacheEnabled           bool
	//BufferedIncomingTxCacheMaxSize           int
	//BufferedIncomingTxCacheExpirationSeconds int
	//FiredTxCacheMaxSize                      int
	//FiredTxCacheExpirationSeconds            int
	//NewTxsChannelSize                        int
}

type Syncer2 struct {
	Config                  *SyncerConfig
	PeerOutgoing            communication.OgPeerCommunicatorOutgoing
	acquireTxQueue          chan *events.NeedSyncEvent
	acquireTxDuplicateCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
	quit                    chan bool
}

func (s *Syncer2) ClearQueue() {
	panic("implement me")
}

func (s *Syncer2) HandlerDescription(et eventbus.EventType) string {
	switch et {
	case events.NeedSyncEventType:
		return "SendSyncRequestToPartner"
	default:
		return "N/A"
	}
}

func (s *Syncer2) HandleEvent(ev eventbus.Event) {
	switch ev.GetEventType() {
	case events.NeedSyncEventType:
		evs := ev.(*events.NeedSyncEvent)
		s.EnqueueRequest(evs)
	default:
		logrus.Warn("event type not supported by syncer2")
	}
}

func (s *Syncer2) InitDefault() {
	s.acquireTxQueue = make(chan *events.NeedSyncEvent)
	s.acquireTxDuplicateCache = gcache.New(s.Config.AcquireTxDedupCacheMaxSize).Simple().
		Expiration(time.Second * time.Duration(s.Config.AcquireTxDedupCacheExpirationSeconds)).Build()
	s.quit = make(chan bool)
}

func (s *Syncer2) Start() {
	go s.loop()
}

func (s *Syncer2) Stop() {
	close(s.quit)
}

func (s *Syncer2) Name() string {
	return "Syncer"
}

func (s *Syncer2) loop() {
	//timer := time.NewTicker(time.Millisecond * time.Duration(s.Config.BatchTimeoutMilliSecond))
	for {
		//var toSend []*SyncRequest
		select {
		//case <- timer.C:
		// send all requests in toSend if available
		case v := <-s.acquireTxQueue:
			// TODO: dedup in acquireTxDuplicateCache
			if v.SpecifiedSource == nil {
				// send this request to all peers
				s.PeerOutgoing.Broadcast(&message.OgMessageBatchSyncRequest{
					Height: nil,
					Hashes: common.Hashes{
						v.Hash,
					},
					BloomFilter: nil,
					RequestId:   0,
				})
			} else {
				// send this request to one peer
				s.PeerOutgoing.Unicast(&message.OgMessageBatchSyncRequest{
					Height: nil,
					Hashes: common.Hashes{
						v.Hash,
					},
					BloomFilter: nil,
					RequestId:   0,
				}, v.SpecifiedSource)
			}
		case <-s.quit:
			break
		}
	}
}

func (s *Syncer2) EnqueueRequest(request *events.NeedSyncEvent) {
	s.acquireTxQueue <- request
}
