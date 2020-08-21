package ogsyncer

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/todolist"
	"github.com/latifrons/goffchan"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const ParallelSyncRequests = 1
const MaxTolerantHeightDiff = 0 // ogsyncer will start syncing if myHeight + MaxTolerantHeightDiff < knownMaxPeerHeight

// RandomPickerContentFetcher will try its best to fetch content,
// until the content is fetched, or timeout, or max try exceeded.
// between each try there will be an interval.
// You should resolve the task explicitly if you get the answer.
type RandomPickerContentFetcher struct {
	ExpireDuration          time.Duration
	MinimumIntervalDuration time.Duration
	MaxTryTimes             int
	peerManager             *PeerManager

	Reporter *soccerdash.Reporter
	taskList *todolist.TodoList

	peerJoinedEventChan           chan *og_interface.PeerJoinedEvent
	newHeightDetectedEventChan    chan *og_interface.NewHeightDetectedEvent
	newHeightBlockSyncedEventChan chan *og_interface.NewHeightBlockSyncedEvent
	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent
	syncTriggerChan               chan string

	quit chan bool
	mu   sync.RWMutex
}

func (b *RandomPickerContentFetcher) InitDefault() {
	peerManager := &PeerManager{
		Reporter: b.Reporter,
	}
	peerManager.InitDefault()
	b.peerManager = peerManager

	b.taskList = &todolist.TodoList{
		ExpireDuration:          b.ExpireDuration,
		MinimumIntervalDuration: b.MinimumIntervalDuration,
		MaxTryTimes:             b.MaxTryTimes,
	}
	b.taskList.InitDefault()

	b.peerJoinedEventChan = make(chan *og_interface.PeerJoinedEvent)
	b.newHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	b.newHeightBlockSyncedEventChan = make(chan *og_interface.NewHeightBlockSyncedEvent)
	b.syncTriggerChan = make(chan string)
	b.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}

	b.quit = make(chan bool)
}

func (b *RandomPickerContentFetcher) PeerLeftChannel() chan *og_interface.PeerLeftEvent {
	panic("implement me")
}

func (b *RandomPickerContentFetcher) NewHeightDetectedEventChannel() chan *og_interface.NewHeightDetectedEvent {
	return b.newHeightDetectedEventChan
}

func (b *RandomPickerContentFetcher) NewHeightBlockSyncedChannel() chan *og_interface.NewHeightBlockSyncedEvent {
	return b.newHeightBlockSyncedEventChan
}

func (b *RandomPickerContentFetcher) NeedToKnow(unknown ogsyncer_interface.Unknown) {
	b.taskList.AddTask(unknown)
	b.triggerSync("NeedToKnow")
}

func (b *RandomPickerContentFetcher) Resolve(unknown ogsyncer_interface.Unknown) {
	b.taskList.RemoveTask(unknown)
}

// notify sending events
func (b *RandomPickerContentFetcher) AddSubscriberNewOutgoingMessageEvent(transport *transport.PhysicalCommunicator) {
	b.newOutgoingMessageSubscribers = append(b.newOutgoingMessageSubscribers, transport)
}

func (b *RandomPickerContentFetcher) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range b.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing ogsyncer"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (b *RandomPickerContentFetcher) PeerJoinedChannel() chan *og_interface.PeerJoinedEvent {
	return b.peerJoinedEventChan
}

func (b *RandomPickerContentFetcher) Start() {
	go b.eventLoop()
	go b.sync()
	go b.keepUpdateHeights()
}

func (b *RandomPickerContentFetcher) Stop() {
	b.quit <- true
}

func (b *RandomPickerContentFetcher) Name() string {
	return "RandomPickerContentFetcher"
}

func (b *RandomPickerContentFetcher) eventLoop() {
	// TODO: currently believe all heights given by others are real
	// in the future only believe height given by committee
	for {
		select {
		case <-b.quit:
			return
		case event := <-b.peerJoinedEventChan:
			// send height request
			b.handlePeerJoinedEvent(event)
		case event := <-b.newHeightDetectedEventChan:
			b.peerManager.updateKnownPeerHeight(event.PeerId, event.Height)
		case event := <-b.newHeightBlockSyncedEventChan:
			b.handleNewHeightBlockSyncedEvent(event)
		}
		b.Reporter.Report("knownHeight", b.peerManager.knownMaxPeerHeight, false)
	}
}

func (b *RandomPickerContentFetcher) sync() {
	//timer := time.NewTicker(time.Second * time.Duration(SyncCheckIntervalSeconds))
	var s string = ""
	for {
		toSync := false
		select {
		case <-b.quit:
			//timer.Stop()
			//utilfuncs.DrainTicker(timer)
			return
		case s = <-b.syncTriggerChan:
			// start sync because we find a higher height or we received a height update
			toSync = true
			//case <-timer.C:
			//	toSync = true
		}
		if !toSync {
			continue
		}
		b.startSyncOnce(s)
	}
}

func (b *RandomPickerContentFetcher) triggerSync(reason string) {
	select {
	case b.syncTriggerChan <- reason:
	default:
	}

}

func (b *RandomPickerContentFetcher) pickUpRandomSourcePeer(height int64) (peerId string, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.peerManager.peerHeights) == 0 {
		err = errors.New("no peer to sync from")
		return
	}
	higherPeers := b.peerManager.getAllHigherPeers(height)

	if len(higherPeers) == 0 {
		err = fmt.Errorf("already been the highest among connected peers: %d", height)
		return
	}
	// pick random one
	return higherPeers[rand.Intn(len(higherPeers))], nil
}

func (b *RandomPickerContentFetcher) startSyncOnce(reason string) {
	// keep consume tasks until there is nothing left.
	for i := 0; i < ParallelSyncRequests; i++ {
		// handle tasks
		b.doOneTask("pccc " + reason)
	}
}

func (b *RandomPickerContentFetcher) doOneTask(reason string) {
	// taskList will keep this task in the queue until a resolve command is fired.
	task := b.taskList.GetTask()
	if task == nil {
		return
	}
	logrus.WithField("task", task.GetId()).WithField("reason", reason).Warn("handling task")
	b.handleSyncTask(task)
}

func (b *RandomPickerContentFetcher) handlePeerJoinedEvent(event *og_interface.PeerJoinedEvent) {
	logrus.WithField("peer", event.PeerId).Warn("peer joined")
	b.peerManager.updateKnownPeerHeight(event.PeerId, 0)
	b.queryHeights([]string{event.PeerId})
}

func (b *RandomPickerContentFetcher) queryHeights(peerIds []string) {
	// query heights
	resp := &ogsyncer_interface.OgSyncLatestHeightRequest{}
	letterOut := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            resp,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   peerIds,
	}
	b.notifyNewOutgoingMessage(letterOut)
}

func (b *RandomPickerContentFetcher) handleSyncTask(task todolist.Traceable) {
	switch task.(ogsyncer_interface.Unknown).GetType() {
	case ogsyncer_interface.UnknownTypeHeight:
		taskv := task.(*ogsyncer_interface.UnknownHeight)
		if b.handleSyncHeightTask(taskv) {
			return
		}
	case ogsyncer_interface.UnknownTypeHash:
		taskv := task.(*ogsyncer_interface.UnknownHash)
		if b.handleSyncHashTask(taskv) {
			return
		}
	}
}

func (b *RandomPickerContentFetcher) handleSyncHeightTask(taskv *ogsyncer_interface.UnknownHeight) bool {
	peerId, err := b.pickUpRandomSourcePeer(taskv.Height)
	if err != nil {
		logrus.WithError(err).Warn("we know a higher nextHeight but we failed to pick up source peer")
		return true
	}
	logrus.WithField("height", taskv.Height).WithField("from", peerId).Info("ask for height")
	// send sync request to this peer
	// always start offset from 0.
	// if there is more, send another request in the response handler function
	req := &ogsyncer_interface.OgSyncBlockByHeightRequest{
		Height: taskv.Height,
		Offset: 0,
	}

	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            req,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{peerId},
	}
	b.notifyNewOutgoingMessage(letter)
	return false
}

func (b *RandomPickerContentFetcher) handleSyncHashTask(taskv *ogsyncer_interface.UnknownHash) bool {
	peerId, err := b.pickUpRandomSourcePeer(0)
	if err != nil {
		logrus.WithError(err).Warn("we failed to pick up source peer")
		return true
	}
	logrus.WithField("hash", taskv.Hash.HashString()).WithField("from", peerId).Info("ask for hash")
	req := &ogsyncer_interface.OgSyncByHashesRequest{
		Hashes: [][]byte{
			taskv.Hash.Bytes(),
		},
	}

	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            req,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{peerId},
	}
	b.notifyNewOutgoingMessage(letter)
	return false
}

func (b *RandomPickerContentFetcher) keepUpdateHeights() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-b.quit:
			return
		case <-ticker.C:
			b.updateHeightOnce()
		}
	}
}

func (b *RandomPickerContentFetcher) updateHeightOnce() {
	peersToUpdate := b.peerManager.findOutdatedPeersToQueryHeight(5)
	if len(peersToUpdate) != 0 {
		b.queryHeights(peersToUpdate)
	}
}

func (b *RandomPickerContentFetcher) handleNewHeightBlockSyncedEvent(event *og_interface.NewHeightBlockSyncedEvent) {
	if event.Height == b.peerManager.knownMaxPeerHeight {
		// trigger another query
		b.updateHeightOnce()
	}
}