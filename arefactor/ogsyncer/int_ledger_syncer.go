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
	newHeightDetectedEventChan     chan *og_interface.NewHeightDetectedEventArg
	newLocalHeightUpdatedEventChan chan *og_interface.NewLocalHeightUpdatedEventArg
	unknownNeededEventChan         chan ogsyncer_interface.Unknown
	intsReceivedEventChan          chan *ogsyncer_interface.IntsReceivedEventArg

	newIncomingMessageEventChan chan *transport_interface.IncomingLetter

	knownMaxPeerHeight int64

	quit chan bool
}

func (s *IntLedgerSyncer) Receive(topic int, msg interface{}) error {
	switch consts.EventType(topic) {
	case consts.NewIncomingMessageEvent:
		s.newIncomingMessageEventChan <- msg.(*transport_interface.IncomingLetter)
	case consts.UnknownNeededEvent:
		s.unknownNeededEventChan <- msg.(ogsyncer_interface.Unknown)
	case consts.IntsReceivedEvent:
		s.intsReceivedEventChan <- msg.(*ogsyncer_interface.IntsReceivedEventArg)
	case consts.NewHeightDetectedEvent:
		s.newHeightDetectedEventChan <- msg.(*og_interface.NewHeightDetectedEventArg)
	default:
		return eventbus.ErrNotSupported
	}
	return nil
}

func (s *IntLedgerSyncer) InitDefault() {
	s.newHeightDetectedEventChan = make(chan *og_interface.NewHeightDetectedEventArg)
	s.newLocalHeightUpdatedEventChan = make(chan *og_interface.NewLocalHeightUpdatedEventArg)
	s.intsReceivedEventChan = make(chan *ogsyncer_interface.IntsReceivedEventArg)

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

func (s *IntLedgerSyncer) notifyNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEventArg) {
	s.EventBus.Publish(int(consts.NewHeightDetectedEvent), event)
}

func (s *IntLedgerSyncer) notifyNewHeightBlockSynced(event *og_interface.ResourceGotEvent) {
	s.EventBus.Publish(int(consts.NewHeightBlockSyncedEvent), event)
}

func (s *IntLedgerSyncer) eventLoop() {
	timer := time.NewTicker(time.Second * time.Duration(SyncCheckHeightIntervalSeconds))
	for {
		logrus.Debug("Another round?")
		select {
		case <-s.quit:
			timer.Stop()
			utilfuncs.DrainTicker(timer)
			return
		case event := <-s.newHeightDetectedEventChan:
			s.handleNewHeightDetectedEvent(event)
		case event := <-s.newLocalHeightUpdatedEventChan:
			s.handleNewLocalHeightUpdatedEvent(event)
		case event := <-s.unknownNeededEventChan:
			switch event.GetType() {
			case ogsyncer_interface.UnknownTypeHash:
				s.enqueueHashTask(event.(*ogsyncer_interface.UnknownHash).Hash, "")
			case ogsyncer_interface.UnknownTypeHeight:
				s.enqueueHeightTask(event.(*ogsyncer_interface.UnknownHeight).Height, "")
			default:
				logrus.Warn("Unexpected unknown type")
			}
		case event := <-s.intsReceivedEventChan:
			s.handleIntsReceivedEvent(event)
		}
	}
}

// messageLoop will separate from event loop since IntLedgerSyncer will generate event
func (s *IntLedgerSyncer) messageLoop() {
	for {
		select {
		case letter := <-s.newIncomingMessageEventChan:
			s.handleIncomingMessage(letter)
		case <-s.quit:
			return
		}
	}
}

func (s *IntLedgerSyncer) handleNewHeightDetectedEvent(event *og_interface.NewHeightDetectedEventArg) {
	s.knownMaxPeerHeight = math.BiggerInt64(s.knownMaxPeerHeight, event.Height)
	// record this peer so that we may sync from it in the future.
	if s.Ledger.CurrentHeight() < s.knownMaxPeerHeight {
		s.enqueueHeightTask(s.Ledger.CurrentHeight()+int64(1), event.PeerId)
	}
}

func (s *IntLedgerSyncer) handleNewLocalHeightUpdatedEvent(event *og_interface.NewLocalHeightUpdatedEventArg) {
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
			logrus.WithError(err).Warn("cannot parse OgSyncLatestHeightRequest")
			return
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
			logrus.WithError(err).Warn("cannot parse OgSyncMessageTypeLatestHeightResponse")
			return
		}
		logrus.WithField("reqHeight", req.MyHeight).Debug("OgSyncMessageTypeLatestHeightResponse")
		s.notifyNewHeightDetectedEvent(&og_interface.NewHeightDetectedEventArg{
			Height: req.MyHeight,
			PeerId: letter.From,
		})
		// self listening will trigger another sync.
	case ogsyncer_interface.OgSyncMessageTypeBlockByHeightRequest:
		req := &ogsyncer_interface.OgSyncBlockByHeightRequest{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Warn("cannot parse OgSyncBlockByHeightRequest")
			return
		}
		logrus.WithField("req", req).Debug("OgSyncMessageTypeBlockByHeightRequest")

		s.handleBlockByHeightRequest(req, letter.From)
	//case ogsyncer_interface.OgSyncMessageTypeBlockByHeightResponse:
	//	req := &ogsyncer_interface.OgSyncBlockByHeightResponse{}
	//	err := req.FromBytes(letter.Msg.ContentBytes)
	//	if err != nil {
	//		logrus.WithError(err).Fatal("block by height response")
	//	}
	//	logrus.WithField("req", req).Debug("OgSyncMessageTypeBlockByHeightResponse")
	//	s.handleBlockByHeightResponse(req, letter.From)
	case ogsyncer_interface.OgSyncMessageTypeByHashesRequest:
		req := &ogsyncer_interface.OgSyncBlockByHashRequest{}
		err := req.FromBytes(letter.Msg.ContentBytes)
		if err != nil {
			logrus.WithError(err).Warn("cannot parse OgSyncBlockByHashRequest")
			return
		}
		logrus.WithField("req", req).Debug("OgSyncBlockByHashRequest")

		s.handleBlockByHashRequest(req, letter.From)
	//case ogsyncer_interface.OgSyncMessageTypeBlockByHashRequest:
	//case ogsyncer_interface.OgSyncMessageTypeByHashesResponse:
	//case ogsyncer_interface.OgSyncMessageTypeByBlockHashResponse:
	default:
	}
}

func (s *IntLedgerSyncer) handleBlockByHeightRequest(req *ogsyncer_interface.OgSyncBlockByHeightRequest, from string) {
	blockContent := s.Ledger.GetBlock(req.Height)
	if blockContent == nil {
		// I don't have this. no response
		logrus.WithField("height", req.Height).Debug("I don't have this block")
		return
	}
	s.sendBlockResponse(blockContent, from)
}

func (s *IntLedgerSyncer) handleBlockByHashRequest(req *ogsyncer_interface.OgSyncBlockByHashRequest, from string) {
	hash := &og_interface.Hash32{}
	hash.FromBytes(req.Hash)
	blockContent := s.Ledger.GetBlockByHash(hash.HashString())
	if blockContent == nil {
		// I don't have this. no response
		logrus.WithField("hash", req.Hash).Debug("I don't have this block")
		return
	}
	s.sendBlockResponse(blockContent, from)

}

func (s *IntLedgerSyncer) sendBlockResponse(blockContent og_interface.BlockContent, from string) {
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

func (s *IntLedgerSyncer) resolveBlock(block *dummy.IntArrayBlockContent, from string) {
	// In normal case, this should be done in TxBuffer and TxPool, conducted by ConsensusEnforcer
	// Here we just bypass all things and insert the block directly to ledger

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
	s.trySyncNextHeight()
	//b.Reporter.Report("tasks", b.taskList)
	//s.notifyNewHeightBlockSynced(&og_interface.ResourceGotEvent{
	//	Height: block.Height,
	//})
}

func (s *IntLedgerSyncer) trySyncNextHeight() {
	// immediately trigger a sync to continuously empty the task queue.
	if s.Ledger.CurrentHeight() < s.knownMaxPeerHeight {
		s.enqueueHeightTask(s.Ledger.CurrentHeight()+int64(1), "handleBlockByHeightResponse")
	}
}

func (s *IntLedgerSyncer) handleIntsReceivedEvent(event *ogsyncer_interface.IntsReceivedEventArg) {
	// resolve this
	s.resolveBlock(s.convertBlock(event.Ints), event.From)
}

func (s *IntLedgerSyncer) convertBlock(v ogsyncer_interface.MessageContentInt) *dummy.IntArrayBlockContent {
	block := &dummy.IntArrayBlockContent{
		Height:      v.Height,
		Step:        v.Step,
		PreviousSum: v.PreviousSum,
		MySum:       v.MySum,
		Submitter:   v.Submitter,
		Ts:          v.Ts,
	}
	return block
}
