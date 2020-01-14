package syncer

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/gcache"
	"time"
)

type SyncRequest struct {
	Hash            common.Hash
	SpecifiedSource *communication.OgPeer
}

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
	acquireTxQueue          chan *SyncRequest
	acquireTxDuplicateCache gcache.Cache // list of hashes that are queried recently. Prevent duplicate requests.
}

func (s *Syncer2) InitDefault() {
	s.acquireTxQueue = make(chan *SyncRequest)
	s.acquireTxDuplicateCache = gcache.New(s.Config.AcquireTxDedupCacheMaxSize).Simple().
		Expiration(time.Second * time.Duration(s.Config.AcquireTxDedupCacheExpirationSeconds)).Build()

}

func (s *Syncer2) Start() {
	go s.loop()
}

func (s *Syncer2) Stop() {
	panic("implement me")
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
			// send this request
			if v.SpecifiedSource == nil {
				s.PeerOutgoing.Broadcast(&message.OgMessageBatchSyncRequest{
					Height: nil,
					Hashes: common.Hashes{
						v.Hash,
					},
					BloomFilter: nil,
					RequestId:   0,
				})
			} else {
				s.PeerOutgoing.Unicast(&message.OgMessageBatchSyncRequest{
					Height: nil,
					Hashes: common.Hashes{
						v.Hash,
					},
					BloomFilter: nil,
					RequestId:   0,
				}, v.SpecifiedSource)
			}

		}
	}
}

func (s *Syncer2) EnqueueRequest(request SyncRequest) {

}
