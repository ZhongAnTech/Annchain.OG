package ogsyncer

import (
	"container/list"
	"errors"
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/math"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const SyncCheckIntervalSeconds int = 10 // max check interval for syncing a height
const MaxTolerantHeightDiff = 0         // ogsyncer will start syncing if myHeight + MaxTolerantHeightDiff < knownMaxPeerHeight

type DefaultUnknownManager struct {
	Unknowns list.List
}

func (d *DefaultUnknownManager) Enqueue(task og_interface.Unknown) {
	d.Unknowns.PushBack(task)
}

type IntSyncer struct {
	Fetcher            ogsyncer_interface.ResourceFetcher
	peerHeights        map[string]int64
	knownMaxPeerHeight int64
	unknownManager     og_interface.UnknownManager

	syncTriggerChan               chan bool
	myNewHeightDetectedEventChan  chan *og_interface.NewHeightDetectedEvent
	myPeerLeftEventChan           chan *og_interface.PeerLeftEvent
	myNewIncomingMessageEventChan chan *transport_interface.IncomingLetter

	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	mu   sync.RWMutex
	quit chan bool
}

// notify sending events
func (b *IntSyncer) AddSubscriberNewOutgoingMessageEvent(transport *transport.PhysicalCommunicator) {
	b.newOutgoingMessageSubscribers = append(b.newOutgoingMessageSubscribers, transport)
}

func (d *IntSyncer) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range d.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing ogsyncer"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (b *IntSyncer) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return b.myNewIncomingMessageEventChan
}

func (b *IntSyncer) NewHeightDetectedEventChannel() chan *og_interface.NewHeightDetectedEvent {
	return b.myNewHeightDetectedEventChan
}

func (b *IntSyncer) EventChannelPeerLeft() chan *og_interface.PeerLeftEvent {
	return b.myPeerLeftEventChan
}

func (b *IntSyncer) InitDefault() {
	b.peerHeights = make(map[string]int64)

	b.syncTriggerChan = make(chan bool)
	b.myNewIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)
	b.myNewHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	b.myPeerLeftEventChan = make(chan *og_interface.PeerLeftEvent)

	b.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}
	b.quit = make(chan bool)
}

func (b *IntSyncer) Start() {
	b.knownMaxPeerHeight = b.Ledger.CurrentHeight()
	go b.eventLoop()
	go b.sync()
}

func (b *IntSyncer) Stop() {
	b.quit <- true
}

func (b *IntSyncer) Name() string {
	return "IntSyncer"
}

func (b *IntSyncer) updateKnownPeerHeight(peerId string, height int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// refresh our target
	b.knownMaxPeerHeight = math.BiggerInt64(b.knownMaxPeerHeight, height)
	b.peerHeights[peerId] = height

}

func (b *IntSyncer) removeKnownPeerHeight(peerId string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// refresh our target
	delete(b.peerHeights, peerId)
}

func (b *IntSyncer) eventLoop() {
	// TODO: currently believe all heights given by others are real
	// in the future only believe height given by committee
	for {
		select {
		case <-b.quit:
			return
		case event := <-b.myNewHeightDetectedEventChan:
			// record this peer so that we may sync from it in the future.
			if event.Height > b.knownMaxPeerHeight {
				b.updateKnownPeerHeight(event.PeerId, event.Height)
				// write or not write (already syncing).
				select {
				case b.syncTriggerChan <- true:
				default:
				}
			}
		case event := <-b.myPeerLeftEventChan:
			// remove this peer from potential peers
			b.removeKnownPeerHeight(event.PeerId)
		case letter := <-b.myNewIncomingMessageEventChan:
			b.handleIncomingMessage(letter)
		}
	}
}

func (b *IntSyncer) needSync() bool {
	return b.knownMaxPeerHeight > b.Ledger.CurrentHeight()+MaxTolerantHeightDiff
}

func (b *IntSyncer) sync() {
	timer := time.NewTimer(time.Second * time.Duration(SyncCheckIntervalSeconds))
	for {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(time.Second * time.Duration(SyncCheckIntervalSeconds))
		toSync := false
		select {
		case <-b.quit:
			return
		case <-b.syncTriggerChan:
			// start sync because we find a higher height or we received a height update
			toSync = true
		case <-timer.C:
			// start sync because of check interval
			toSync = true
		}

		if !toSync {
			continue
		}
		logrus.WithFields(logrus.Fields{
			"myHeight":           b.Ledger.CurrentHeight(),
			"knownMaxPeerHeight": b.knownMaxPeerHeight,
		}).Debug("start sync")
		b.startSyncOnce()
	}
}

func (b *IntSyncer) pickUpRandomSourcePeer(height int64) (peerId string, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.peerHeights) == 0 {
		err = errors.New("no peer to sync from")
		return
	}

	higherPeers := []string{}
	for peerId, theirHeight := range b.peerHeights {
		if theirHeight >= height {
			higherPeers = append(higherPeers, peerId)
		}
	}
	if len(higherPeers) == 0 {
		err = errors.New("already been the highest among connected peers")
		return
	}
	// pick random one
	return higherPeers[rand.Intn(len(higherPeers))], nil
}

func (b *IntSyncer) startSyncOnce() {
	if !b.needSync() {
		logrus.Debug("no need to sync")
		return
	}
	// send one sync message to random one
	height := b.Ledger.CurrentHeight() + 1
	peerId, err := b.pickUpRandomSourcePeer(height)
	if err != nil {
		logrus.WithError(err).Warn("we known a higher height but we failed to pick up source peer")
		return
	}

	// send sync request to this peer
	req := &message.OgMessageHeightSyncRequest{
		Height:      height,
		Offset:      0,
		BloomFilter: nil,
	}

	letter := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            req,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{peerId},
	}
	b.notifyNewOutgoingMessage(letter)
}

func (b *IntSyncer) handleIncomingMessage(letter *transport_interface.IncomingLetter) {
	switch message.OgMessageType(letter.Msg.MsgType) {
	case message.OgMessageTypeHeightRequest:
	case message.OgMessageTypeHeightResponse:
	case message.OgMessageTypeResourceRequest:
	case message.OgMessageTypeResourceResponse:
		//
		//	m := &message.OgMessageHeightSyncRequest{}
		//	err := m.FromBytes(letter.Msg.ContentBytes)
		//	if err != nil {
		//		logrus.WithField("type", "OgMessageHeightSyncRequest").WithError(err).Warn("bad message")
		//	}
		//	logrus.WithField("height", m.Height).WithField("from", letter.From).Info("received sync request")
		//
		//	//TODO: fetch data and return
		//	value := b.Ledger.GetBlock(m.Height)
		//	if value == nil {
		//		return
		//	}
		//
		//	content := value.(*dummy.IntArrayBlockContent)
		//
		//	messageContent := &ogsyncer_interface.MessageContentInt{
		//		Height:      content.Height,
		//		Step:        content.Step,
		//		PreviousSum: content.PreviousSum,
		//		MySum:       content.MySum,
		//		Submitter:   content.Submitter,
		//	}
		//
		//	// inner data
		//	resp := &message.OgMessageHeightSyncResponse{
		//		Height:      m.Height,
		//		Offset:      0,
		//		HasNextPage: false,
		//		Resources: []message.MessageContentResource{
		//			{
		//				ResourceType:    ogsyncer_interface.ResourceTypeInt,
		//				ResourceContent: messageContent.ToBytes(),
		//			},
		//		},
		//	}
		//	// letter
		//	letterOut := &transport_interface.OutgoingLetter{
		//		ExceptMyself:   true,
		//		Msg:            resp,
		//		SendType:       transport_interface.SendTypeUnicast,
		//		CloseAfterSent: false,
		//		EndReceivers:   []string{letter.From},
		//	}
		//
		//	b.notifyNewOutgoingMessage(letterOut)
		//
		//case message.OgMessageTypeHeightSyncResponse:
		//
		//	m := &message.OgMessageHeightSyncResponse{}
		//	err := m.FromBytes(letter.Msg.ContentBytes)
		//	if err != nil {
		//		logrus.WithField("type", "OgMessageHeightSyncResponse").WithError(err).Warn("bad message")
		//	}
		//	logrus.WithField("from", letter.From).Info("received height sync response")
		//
		//	// TODO: solve this height and start the next
		//	// TODO: verify signature
		//	for _, resource := range m.Resources {
		//		if resource.ResourceType != ogsyncer_interface.ResourceTypeInt {
		//			panic("invalid resource type")
		//		}
		//		messageContent := &ogsyncer_interface.MessageContentInt{}
		//		err := messageContent.FromBytes(resource.ResourceContent)
		//		utilfuncs.PanicIfError(err, "parse content")
		//
		//		bc := &dummy.IntArrayBlockContent{
		//			Height:      m.Height,
		//			Step:        messageContent.Step,
		//			PreviousSum: messageContent.PreviousSum,
		//			MySum:       messageContent.MySum,
		//		}
		//		b.Ledger.ConfirmBlock(bc)
		//		// TODO: announce event
		//	}
		//	logrus.WithField("height", m.Height).Info("height updated")
		//
		//	select {
		//	case b.syncTriggerChan <- true:
		//	default:
		//	}
	}
}
