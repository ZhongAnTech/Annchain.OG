package ogsyncer

import (
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/math"
	"github.com/annchain/commongo/utilfuncs"
	"github.com/latifrons/goffchan"
	"github.com/sirupsen/logrus"
	"time"
)

const SyncCheckHeightIntervalSeconds int = 1 // max check interval for syncing a content

// IntLedgerSyncer try to restore and update the ledger
// Manage the unknowns
type IntLedgerSyncer struct {
	Ledger og_interface.Ledger

	ContentFetcher                 *RandomPickerContentFetcher
	newHeightDetectedEventChan     chan *og_interface.NewHeightDetectedEvent
	newLocalHeightUpdatedEventChan chan *og_interface.NewLocalHeightUpdatedEvent
	unknownNeededEventChan         chan ogsyncer_interface.Unknown
	newIncomingMessageEventChan    chan *transport_interface.IncomingLetter

	newOutgoingMessageSubscribers        []transport_interface.NewOutgoingMessageEventSubscriber // a message need to be sent
	newHeightDetectedEventSubscribers    []og_interface.NewHeightDetectedEventSubscriber
	newHeightBlockSyncedEventSubscribers []og_interface.ResourceGotEventSubscriber

	knownMaxPeerHeight int64

	quit chan bool
}

func (b *IntLedgerSyncer) InitDefault() {
	b.newHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	b.newLocalHeightUpdatedEventChan = make(chan *og_interface.NewLocalHeightUpdatedEvent)
	b.unknownNeededEventChan = make(chan ogsyncer_interface.Unknown)
	b.newIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)

	// self event registration
	b.AddSubscriberNewHeightDetectedEvent(b)

	b.quit = make(chan bool)
}

func (s *IntLedgerSyncer) Start() {
	go s.eventLoop()
	go s.messageLoop()
}

func (s *IntLedgerSyncer) Stop() {
	s.quit <- true
}

func (s *IntLedgerSyncer) Name() string {
	return "IntLedgerSyncer"
}

func (s *IntLedgerSyncer) NewHeightDetectedEventChannel() chan *og_interface.NewHeightDetectedEvent {
	return s.newHeightDetectedEventChan
}

func (b *IntLedgerSyncer) UnknownNeededEventChannel() chan ogsyncer_interface.Unknown {
	return b.unknownNeededEventChan
}
func (b *IntLedgerSyncer) NewLocalHeightUpdatedChannel() chan *og_interface.NewLocalHeightUpdatedEvent {
	return b.newLocalHeightUpdatedEventChan
}
func (b *IntLedgerSyncer) NewIncomingMessageEventChannel() chan *transport_interface.IncomingLetter {
	return b.newIncomingMessageEventChan
}

// notify sending events
func (b *IntLedgerSyncer) AddSubscriberNewOutgoingMessageEvent(transport *transport.PhysicalCommunicator) {
	b.newOutgoingMessageSubscribers = append(b.newOutgoingMessageSubscribers, transport)
}

func (b *IntLedgerSyncer) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	for _, subscriber := range b.newOutgoingMessageSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewOutgoingMessageEventChannel(), event, "outgoing ogsyncer"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}
func (d *IntLedgerSyncer) AddSubscriberNewHeightDetectedEvent(sub og_interface.NewHeightDetectedEventSubscriber) {
	d.newHeightDetectedEventSubscribers = append(d.newHeightDetectedEventSubscribers, sub)
}

func (n *IntLedgerSyncer) notifyNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEvent) {
	for _, subscriber := range n.newHeightDetectedEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewHeightDetectedEventChannel(), event,
			"notifyNewHeightDetectedEvent"+subscriber.Name()).C
		//subscriber.NewHeightDetectedChannel() <- event
	}
}

// notify sending events
func (b *IntLedgerSyncer) AddSubscriberNewHeightBlockSyncedEvent(sub og_interface.ResourceGotEventSubscriber) {
	b.newHeightBlockSyncedEventSubscribers = append(b.newHeightBlockSyncedEventSubscribers, sub)
}

func (b *IntLedgerSyncer) notifyNewHeightBlockSynced(event *og_interface.ResourceGotEvent) {
	for _, subscriber := range b.newHeightBlockSyncedEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.ResourceGotEventChannel(), event, "notifyNewOutgoingMessage").C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (s *IntLedgerSyncer) eventLoop() {
	timer := time.NewTicker(time.Second * time.Duration(SyncCheckHeightIntervalSeconds))
	for {
		logrus.Warn("Another round?")
		select {
		case <-s.quit:
			timer.Stop()
			utilfuncs.DrainTicker(timer)
			return
		case event := <-s.newHeightDetectedEventChan:
			logrus.WithField("event", event).Info("handleNewHeightDetectedEvent")
			logrus.Info("111")
			s.handleNewHeightDetectedEvent(event)
			logrus.Info("111OK")
		case event := <-s.newLocalHeightUpdatedEventChan:
			logrus.WithField("event", event).Info("handleNewLocalHeightUpdatedEvent")
			logrus.Info("222")
			s.handleNewLocalHeightUpdatedEvent(event)
			logrus.Info("222OK")
		case event := <-s.unknownNeededEventChan:
			logrus.Info("333")
			logrus.WithField("event", event).Info("unknownNeededEventChan")
			switch event.GetType() {
			case ogsyncer_interface.UnknownTypeHash:
				logrus.Fatal("under construction")
				s.enqueueHashTask(event.(*ogsyncer_interface.UnknownHash).Hash, "")
			case ogsyncer_interface.UnknownTypeHeight:
				s.enqueueHeightTask(event.(*ogsyncer_interface.UnknownHeight).Height, "")
			default:
				logrus.Warn("Unexpected unknown type")
			}
			logrus.Info("333OK")
		}
	}
}

// messageLoop will separate from event loop since IntLedgerSyncer will generate event
func (b *IntLedgerSyncer) messageLoop() {
	for {
		select {
		case letter := <-b.newIncomingMessageEventChan:

			//c1 := make(chan bool)
			go func() {
				b.handleIncomingMessage(letter)
				//c1 <- true
			}()

			//select {
			//case <-c1:
			//	break
			//case <-time.After(6 * time.Second):
			//	logrus.Fatal("timeout " + letter.String())
			//}
		case <-b.quit:
			return
		}
	}
}

func (b *IntLedgerSyncer) handleNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEvent) {
	b.knownMaxPeerHeight = math.BiggerInt64(b.knownMaxPeerHeight, event.Height)
	// record this peer so that we may sync from it in the future.
	if b.Ledger.CurrentHeight() < b.knownMaxPeerHeight {
		b.enqueueHeightTask(b.Ledger.CurrentHeight()+int64(1), event.PeerId)
	}
}

func (b *IntLedgerSyncer) handleNewLocalHeightUpdatedEvent(event *og_interface.NewLocalHeightUpdatedEvent) {
	// check height
	if event.Height < b.knownMaxPeerHeight {
		// sync next
		b.enqueueHeightTask(event.Height+int64(1), "")
	}
}

func (b *IntLedgerSyncer) enqueueHashTask(hash og_interface.Hash, hintPeerId string) {
	// sync next
	b.ContentFetcher.NeedToKnow(&ogsyncer_interface.UnknownHash{
		Hash:       hash,
		HintPeerId: hintPeerId,
	})
}

func (b *IntLedgerSyncer) resolveHashTask(hash og_interface.Hash) {
	// sync next
	b.ContentFetcher.Resolve(ogsyncer_interface.UnknownHash{
		Hash: hash,
	})
}

func (b *IntLedgerSyncer) enqueueHeightTask(height int64, hintPeerId string) {
	// sync next
	b.ContentFetcher.NeedToKnow(&ogsyncer_interface.UnknownHeight{
		Height:     height,
		HintPeerId: hintPeerId,
	})
}

func (b *IntLedgerSyncer) resolveHeightTask(height int64) {
	// sync next
	b.ContentFetcher.Resolve(ogsyncer_interface.UnknownHeight{
		Height: height,
	})
}

func (b *IntLedgerSyncer) handleIncomingMessage(letter *transport_interface.IncomingLetter) {
	logrus.WithField("type", letter.Msg.MsgType).Info("Message")
	switch ogsyncer_interface.OgSyncMessageType(letter.Msg.MsgType) {
	case ogsyncer_interface.OgSyncMessageTypeLatestHeightRequest:
		req := &ogsyncer_interface.OgSyncLatestHeightRequest{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("height request")
		}
		// response him a height
		resp := &ogsyncer_interface.OgSyncLatestHeightResponse{
			MyHeight: b.Ledger.CurrentHeight(),
		}
		letterOut := &transport_interface.OutgoingLetter{
			ExceptMyself:   true,
			Msg:            resp,
			SendType:       transport_interface.SendTypeUnicast,
			CloseAfterSent: false,
			EndReceivers:   []string{letter.From},
		}
		logrus.WithField("respHeight", resp.MyHeight).Debug("OgSyncMessageTypeLatestHeightRequest")
		b.notifyNewOutgoingMessage(letterOut)
	case ogsyncer_interface.OgSyncMessageTypeLatestHeightResponse:
		req := &ogsyncer_interface.OgSyncLatestHeightResponse{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("height response")
		}
		logrus.WithField("reqHeight", req.MyHeight).Debug("OgSyncMessageTypeLatestHeightResponse")
		b.notifyNewHeightDetectedEvent(&og_interface.NewHeightDetectedEvent{
			Height: req.MyHeight,
			PeerId: letter.From,
		})
		// self listening will trigger another sync.
	case ogsyncer_interface.OgSyncMessageTypeBlockByHeightRequest:
		req := &ogsyncer_interface.OgSyncBlockByHeightRequest{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("block by height request")
		}
		logrus.WithField("req", req).Debug("OgSyncMessageTypeBlockByHeightRequest")

		b.handleBlockByHeightRequest(req, letter.From)
	case ogsyncer_interface.OgSyncMessageTypeBlockByHeightResponse:
		req := &ogsyncer_interface.OgSyncBlockByHeightResponse{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("block by height response")
		}
		logrus.WithField("req", req).Debug("OgSyncMessageTypeBlockByHeightResponse")
		b.handleBlockByHeightResponse(req, letter.From)
	//case ogsyncer_interface.OgSyncMessageTypeByHashesRequest:
	//case ogsyncer_interface.OgSyncMessageTypeBlockByHashRequest:
	//case ogsyncer_interface.OgSyncMessageTypeByHashesResponse:
	//case ogsyncer_interface.OgSyncMessageTypeByBlockHashResponse:
	default:
	}
}

func (b *IntLedgerSyncer) handleBlockByHeightRequest(req *ogsyncer_interface.OgSyncBlockByHeightRequest, from string) {
	blockContent := b.Ledger.GetBlock(req.Height)
	iab := blockContent.(*dummy.IntArrayBlockContent)

	resp := &ogsyncer_interface.OgSyncBlockByHeightResponse{
		HasMore:    false,
		Sequencers: nil,
		Ints: []ogsyncer_interface.MessageContentInt{
			{
				Height:      iab.Height,
				Step:        iab.Step,
				PreviousSum: iab.PreviousSum,
				MySum:       iab.MySum,
				Submitter:   iab.Submitter,
				Ts:          iab.Ts,
			},
		},
		Txs: nil,
	}
	letterOut := &transport_interface.OutgoingLetter{
		ExceptMyself:   true,
		Msg:            resp,
		SendType:       transport_interface.SendTypeUnicast,
		CloseAfterSent: false,
		EndReceivers:   []string{from},
	}
	b.notifyNewOutgoingMessage(letterOut)
}

func (b *IntLedgerSyncer) handleBlockByHeightResponse(req *ogsyncer_interface.OgSyncBlockByHeightResponse, from string) {
	// do ints
	for _, v := range req.Ints {
		block := &dummy.IntArrayBlockContent{
			Height:      v.Height,
			Step:        v.Step,
			PreviousSum: v.PreviousSum,
			MySum:       v.MySum,
			Submitter:   v.Submitter,
			Ts:          v.Ts,
		}
		b.resolveBlock(block)
	}
	b.trySyncNextHeight()
}

func (b *IntLedgerSyncer) resolveBlock(block *dummy.IntArrayBlockContent, from string) {
	b.Ledger.ConfirmBlock(block)

	// clear all related tasks
	b.resolveHashTask(block.GetHash())
	b.resolveHeightTask(block.GetHeight())

	// in the block-tx env, enrich the txs so that we will continue sync the parents.
	logrus.WithFields(logrus.Fields{
		"from":   from,
		"height": block.Height,
		"hash":   block.GetHash().HashString(),
	}).Trace("got block")
	//b.Reporter.Report("tasks", b.taskList)
	b.notifyNewHeightBlockSynced(&og_interface.ResourceGotEvent{
		Height: block.Height,
	})
}

func (b *IntLedgerSyncer) trySyncNextHeight() {
	// immediately trigger a sync to continuously empty the task queue.
	if b.Ledger.CurrentHeight() < b.knownMaxPeerHeight {
		b.enqueueHeightTask(b.Ledger.CurrentHeight()+int64(1), "handleBlockByHeightResponse")
	}
}
