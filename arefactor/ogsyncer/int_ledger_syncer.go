package ogsyncer

import (
	"github.com/annchain/OG/arefactor/consts"
	"github.com/annchain/OG/arefactor/dummy"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/math"
	"github.com/annchain/commongo/utilfuncs"
	"github.com/latifrons/go-eventbus"
	"github.com/sirupsen/logrus"
	"time"
)

const SyncCheckHeightIntervalSeconds int = 1 // max check interval for syncing a content

// IntLedgerSyncer try to restore and update the ledger
// Manage the unknowns
type IntLedgerSyncer struct {
	EventBus *eventbus.EventBus
	Ledger   og_interface.Ledger

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

func (s *IntLedgerSyncer) Receive(topic int, msg interface{}) error {
	switch consts.EventType(topic) {
	case consts.NewIncomingMessageEvent:
		s.newIncomingMessageEventChan <- msg.(*transport_interface.IncomingLetter)
	case consts.UnknownNeededEvent:
		s.unknownNeededEventChan <- msg.(ogsyncer_interface.Unknown)
	default:
		return eventbus.ErrNotSupported
	}
	return nil
}

func (s *IntLedgerSyncer) InitDefault() {
	s.newHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEvent)
	s.newLocalHeightUpdatedEventChan = make(chan *og_interface.NewLocalHeightUpdatedEvent)
	s.unknownNeededEventChan = make(chan ogsyncer_interface.Unknown)
	s.newIncomingMessageEventChan = make(chan *transport_interface.IncomingLetter)

	s.quit = make(chan bool)
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

func (s *IntLedgerSyncer) notifyNewOutgoingMessage(event *transport_interface.OutgoingLetter) {
	s.EventBus.Publish(int(consts.NewOutgoingMessageEvent), event)
}

func (s *IntLedgerSyncer) notifyNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEvent) {
	s.EventBus.Publish(int(consts.NewHeightDetectedEvent), event)
}

func (s *IntLedgerSyncer) notifyNewHeightBlockSynced(event *og_interface.ResourceGotEvent) {
	s.EventBus.Publish(int(consts.NewHeightBlockSyncedEvent), event)
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
func (s *IntLedgerSyncer) messageLoop() {
	for {
		select {
		case letter := <-s.newIncomingMessageEventChan:

			//c1 := make(chan bool)
			go func() {
				s.handleIncomingMessage(letter)
				//c1 <- true
			}()

			//select {
			//case <-c1:
			//	break
			//case <-time.After(6 * time.Second):
			//	logrus.Fatal("timeout " + letter.String())
			//}
		case <-s.quit:
			return
		}
	}
}

func (s *IntLedgerSyncer) handleNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEvent) {
	s.knownMaxPeerHeight = math.BiggerInt64(s.knownMaxPeerHeight, event.Height)
	// record this peer so that we may sync from it in the future.
	if s.Ledger.CurrentHeight() < s.knownMaxPeerHeight {
		s.enqueueHeightTask(s.Ledger.CurrentHeight()+int64(1), event.PeerId)
	}
}

func (s *IntLedgerSyncer) handleNewLocalHeightUpdatedEvent(event *og_interface.NewLocalHeightUpdatedEvent) {
	// check height
	if event.Height < s.knownMaxPeerHeight {
		// sync next
		s.enqueueHeightTask(event.Height+int64(1), "")
	}
}

func (s *IntLedgerSyncer) enqueueHashTask(hash og_interface.Hash, hintPeerId string) {
	// sync next
	s.ContentFetcher.NeedToKnow(&ogsyncer_interface.UnknownHash{
		Hash:       hash,
		HintPeerId: hintPeerId,
	})
}

func (s *IntLedgerSyncer) resolveHashTask(hash og_interface.Hash) {
	// sync next
	s.ContentFetcher.Resolve(ogsyncer_interface.UnknownHash{
		Hash: hash,
	})
}

func (s *IntLedgerSyncer) enqueueHeightTask(height int64, hintPeerId string) {
	// sync next
	s.ContentFetcher.NeedToKnow(&ogsyncer_interface.UnknownHeight{
		Height:     height,
		HintPeerId: hintPeerId,
	})
}

func (s *IntLedgerSyncer) resolveHeightTask(height int64) {
	// sync next
	s.ContentFetcher.Resolve(ogsyncer_interface.UnknownHeight{
		Height: height,
	})
}

func (s *IntLedgerSyncer) handleIncomingMessage(letter *transport_interface.IncomingLetter) {
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
			MyHeight: s.Ledger.CurrentHeight(),
		}
		letterOut := &transport_interface.OutgoingLetter{
			ExceptMyself:   true,
			Msg:            resp,
			SendType:       transport_interface.SendTypeUnicast,
			CloseAfterSent: false,
			EndReceivers:   []string{letter.From},
		}
		logrus.WithField("respHeight", resp.MyHeight).Debug("OgSyncMessageTypeLatestHeightRequest")
		s.notifyNewOutgoingMessage(letterOut)
	case ogsyncer_interface.OgSyncMessageTypeLatestHeightResponse:
		req := &ogsyncer_interface.OgSyncLatestHeightResponse{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("height response")
		}
		logrus.WithField("reqHeight", req.MyHeight).Debug("OgSyncMessageTypeLatestHeightResponse")
		s.notifyNewHeightDetectedEvent(&og_interface.NewHeightDetectedEvent{
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

		s.handleBlockByHeightRequest(req, letter.From)
	case ogsyncer_interface.OgSyncMessageTypeBlockByHeightResponse:
		req := &ogsyncer_interface.OgSyncBlockByHeightResponse{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Fatal("block by height response")
		}
		logrus.WithField("req", req).Debug("OgSyncMessageTypeBlockByHeightResponse")
		s.handleBlockByHeightResponse(req, letter.From)
	//case ogsyncer_interface.OgSyncMessageTypeByHashesRequest:
	//case ogsyncer_interface.OgSyncMessageTypeBlockByHashRequest:
	//case ogsyncer_interface.OgSyncMessageTypeByHashesResponse:
	//case ogsyncer_interface.OgSyncMessageTypeByBlockHashResponse:
	default:
	}
}

func (s *IntLedgerSyncer) handleBlockByHeightRequest(req *ogsyncer_interface.OgSyncBlockByHeightRequest, from string) {
	blockContent := s.Ledger.GetBlock(req.Height)
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
	s.notifyNewOutgoingMessage(letterOut)
}

func (s *IntLedgerSyncer) handleBlockByHeightResponse(req *ogsyncer_interface.OgSyncBlockByHeightResponse, from string) {
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
		s.resolveBlock(block)
	}
	s.trySyncNextHeight()
}

func (s *IntLedgerSyncer) resolveBlock(block *dummy.IntArrayBlockContent, from string) {
	s.Ledger.ConfirmBlock(block)

	// clear all related tasks
	s.resolveHashTask(block.GetHash())
	s.resolveHeightTask(block.GetHeight())

	// in the block-tx env, enrich the txs so that we will continue sync the parents.
	logrus.WithFields(logrus.Fields{
		"from":   from,
		"height": block.Height,
		"hash":   block.GetHash().HashString(),
	}).Trace("got block")
	//b.Reporter.Report("tasks", b.taskList)
	s.notifyNewHeightBlockSynced(&og_interface.ResourceGotEvent{
		Height: block.Height,
	})
}

func (s *IntLedgerSyncer) trySyncNextHeight() {
	// immediately trigger a sync to continuously empty the task queue.
	if s.Ledger.CurrentHeight() < s.knownMaxPeerHeight {
		s.enqueueHeightTask(s.Ledger.CurrentHeight()+int64(1), "handleBlockByHeightResponse")
	}
}
