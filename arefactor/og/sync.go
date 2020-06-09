package og

import (
	"container/list"
	"github.com/annchain/OG/arefactor/og_interface"
	"sync"
	"time"
)

const SyncCheckIntervalSeconds int = 5

type DefaultUnknownManager struct {
	Unknowns list.List
}

func (d *DefaultUnknownManager) Enqueue(task og_interface.Unknown) {
	d.Unknowns.PushBack(task)
}

type BlockByBlockSyncer struct {
	Ledger                       Ledger
	knownMaxHeight               int64
	knownMaxHeightPeerId         string
	unknownManager               og_interface.UnknownManager
	myNewHeightDetectedEventChan chan *og_interface.NewHeightDetectedEvent

	mu   sync.Mutex
	quit chan bool
}

func (b *BlockByBlockSyncer) InitDefault() {
	b.myNewHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	b.quit = make(chan bool)
}

func (b *BlockByBlockSyncer) Start() {
	b.knownMaxHeight = b.Ledger.CurrentHeight()
	go b.eventLoop()
	go b.sync()
}

func (b *BlockByBlockSyncer) Stop() {
	b.quit <- true
}

func (b *BlockByBlockSyncer) Name() string {
	return "BlockByBlockSyncer"
}

func (b *BlockByBlockSyncer) eventLoop() {
	// TODO: currently believe all heights given by others are real
	// in the future only believe height given by committee
	timer := time.NewTimer(time.Second * time.Duration(SyncCheckIntervalSeconds))
	for {
		if !timer.Stop() {
			<-timer.C
		}
		timer.Reset(time.Second * time.Duration(SyncCheckIntervalSeconds))
		select {
		case event := <-b.myNewHeightDetectedEventChan:
			// record this peer so that we may sync from it in the future.
			if event.Height > b.knownMaxHeight {
				b.mu.Lock()
				// refresh our target
				b.knownMaxHeight = event.Height
				b.knownMaxHeightPeerId = event.PeerId
				b.mu.Unlock()
			}
		case <-timer.C:

		}
	}
}

func (b *BlockByBlockSyncer) sync() {
	for {
		select {
		//case <-time.Tick()
		}
	}
}
