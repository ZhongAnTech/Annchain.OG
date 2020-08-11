package ogsyncer

import (
	"errors"
	"github.com/annchain/OG/arefactor/og/message"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/utilfuncs"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const SyncCheckIntervalSeconds int = 10 // max check interval for syncing a height
const MaxTolerantHeightDiff = 0         // ogsyncer will start syncing if myHeight + MaxTolerantHeightDiff < knownMaxPeerHeight

// IntSyncer2 listens to multiple event channels to judge what to do.
// On new height detected: compare to local and see if we need to sync from it
// On new incoming message: handle sync request and response.
// On peer left: switch peer to sync?
// handle unknown resource. ask others.
type IntSyncer2 struct {
	Ledger         og_interface.Ledger
	unknownManager ogsyncer_interface.UnknownManager
	peerManager    PeerManager

	newHeightDetectedEventChan  chan *og_interface.NewHeightDetectedEvent
	peerJoinedEventChan         chan *og_interface.PeerJoinedEvent
	newIncomingMessageEventChan chan *transport_interface.IncomingLetter

	newOutgoingMessageSubscribers []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent

	syncTriggerChan chan bool

	quit chan bool
	mu   sync.RWMutex
}

func (b *IntSyncer2) NeedToKnow(unknown ogsyncer_interface.Unknown, hint ogsyncer_interface.SourceHint) {
	b.unknownManager.Enqueue(unknown, hint)
}

// notify sending events
func (b *IntSyncer2) AddSubscriberNewOutgoingMessageEvent(transport *transport.PhysicalCommunicator) {
	b.newOutgoingMessageSubscribers = append(b.newOutgoingMessageSubscribers, transport)
}

func (d *IntSyncer2) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range d.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing ogsyncer"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (b *IntSyncer2) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return b.newIncomingMessageEventChan
}

func (b *IntSyncer2) NewHeightDetectedEventChannel() chan *og_interface.NewHeightDetectedEvent {
	return b.newHeightDetectedEventChan
}

func (b *IntSyncer2) EventChannelPeerJoinedChannel() chan *og_interface.PeerJoinedEvent {
	return b.peerJoinedEventChan
}

func (b *IntSyncer2) InitDefault() {
	unknownManager := &DefaultUnknownManager{}
	unknownManager.InitDefault()
	b.unknownManager = unknownManager
	b.newIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)
	b.newHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	b.peerJoinedEventChan = make(chan *og_interface.PeerJoinedEvent)

	b.newOutgoingMessageSubscribers = []transport_interface.NewOutgoingMessageEventSubscriber{}
	b.quit = make(chan bool)
}

func (b *IntSyncer2) Start() {
	b.peerManager.myHeight = b.Ledger.CurrentHeight()
	go b.eventLoop()
}

func (b *IntSyncer2) Stop() {
	b.quit <- true
}

func (b *IntSyncer2) Name() string {
	return "IntSyncer2"
}

func (b *IntSyncer2) eventLoop() {
	// TODO: currently believe all heights given by others are real
	// in the future only believe height given by committee
	for {
		select {
		case <-b.quit:
			return
		case event := <-b.newHeightDetectedEventChan:
			b.handleNewHeightDetectedEvent(event)
		case event := <-b.peerJoinedEventChan:
			// send height request
			b.handlePeerJoinedEvent(event)
		case letter := <-b.newIncomingMessageEventChan:
			b.handleIncomingMessage(letter)
		}
	}
}

func (b *IntSyncer2) needSync() bool {
	return b.peerManager.knownMaxPeerHeight > b.Ledger.CurrentHeight()+MaxTolerantHeightDiff
}

func (b *IntSyncer2) sync() {
	timer := time.NewTimer(time.Second * time.Duration(SyncCheckIntervalSeconds))
	for {
		utilfuncs.DrainTimer(timer)
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
			"knownMaxPeerHeight": b.peerManager.knownMaxPeerHeight,
		}).Debug("start sync")
		b.startSyncOnce()
	}
}

func (b *IntSyncer2) pickUpRandomSourcePeer(height int64) (peerId string, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.peerManager.peerHeights) == 0 {
		err = errors.New("no peer to sync from")
		return
	}
	higherPeers := b.peerManager.getAllHigherPeers(height)

	if len(higherPeers) == 0 {
		err = errors.New("already been the highest among connected peers")
		return
	}
	// pick random one
	return higherPeers[rand.Intn(len(higherPeers))], nil
}

func (b *IntSyncer2) startSyncOnce() {
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
	// always start offset from 0.
	// if there is more, send another request in the response handler function
	req := &ogsyncer_interface.OgSyncBlockByHeightRequest{
		Height: height,
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
}

func (b *IntSyncer2) handleIncomingMessage(letter *transport_interface.IncomingLetter) {
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

func (b *IntSyncer2) handlePeerJoinedEvent(event *og_interface.PeerJoinedEvent) {

}

func (b *IntSyncer2) handleNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEvent) {
	// record this peer so that we may sync from it in the future.
	if event.Height > b.peerManager.knownMaxPeerHeight {
		b.peerManager.updateKnownPeerHeight(event.PeerId, event.Height)
		// write or not write (already syncing).
		select {
		case b.syncTriggerChan <- true:
		default:
		}
	}
}
