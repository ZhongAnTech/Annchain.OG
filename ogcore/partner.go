package ogcore

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/debug/debuglog"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/interfaces"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/OG/ogcore/model"
	"sync"
)

// OgPartner handles messages coming from the outside. It is the ingress of internal core.
// Focus on I/O. Any modules that can be run under solo mode should be in ogcore.
type OgPartner struct {
	debuglog.NodeLogger
	Config       OgProcessorConfig
	PeerOutgoing communication.OgPeerCommunicatorOutgoing
	PeerIncoming communication.OgPeerCommunicatorIncoming
	EventBus     eventbus.EventBus

	// og protocols
	StatusProvider interfaces.OgStatusProvider
	OgCore         *OgCore
	Syncer         interfaces.Syncer

	// partner should connect the backend service by announce events
	quit   chan bool
	quitWg sync.WaitGroup
}

func (op *OgPartner) AdaptTxiToResource(txi types.Txi) message.MessageContentResource {
	switch txi.GetType() {
	case types.TxBaseTypeTx:
		tx := txi.(*types.Tx)
		content := message.MessageContentTx{
			Hash:         tx.Hash,
			ParentsHash:  tx.ParentsHash,
			MineNonce:    tx.MineNonce,
			AccountNonce: tx.AccountNonce,
			From:         tx.From,
			To:           tx.To,
			Value:        tx.Value,
			TokenId:      tx.TokenId,
			PublicKey:    tx.PublicKey.ToBytes(),
			Data:         tx.Data,
			Signature:    tx.Signature.ToBytes(),
		}
		return message.MessageContentResource{
			ResourceType:    message.ResourceTypeTx,
			ResourceContent: content.ToBytes(),
		}
	case types.TxBaseTypeSequencer:
		seq := txi.(*types.Sequencer)
		content := message.MessageContentSequencer{
			Hash:         seq.Hash,
			ParentsHash:  seq.ParentsHash,
			MineNonce:    seq.MineNonce,
			AccountNonce: seq.AccountNonce,
			Issuer:       seq.Issuer,
			PublicKey:    seq.PublicKey,
			Signature:    seq.Signature,
			StateRoot:    seq.StateRoot,
			Height:       seq.Height,
		}
		return message.MessageContentResource{
			ResourceType:    message.ResourceTypeSequencer,
			ResourceContent: content.ToBytes(),
		}
	default:
		panic("Should not be here. Do you miss adding tx type?")
	}
}

func (op *OgPartner) HandleEvent(ev eventbus.Event) {
	// handle sending events
	switch ev.GetEventType() {
	case events.HeightSyncRequestReceivedEventType:
		evt := ev.(*events.HeightSyncRequestReceivedEvent)
		txs := op.OgCore.LoadHeightTxis(evt.Height, evt.Offset, op.Config.MaxTxCountInResponse)
		op.EventBus.Route(&events.TxsFetchedForResponseEvent{
			Txs:       txs,
			RequestId: evt.RequestId,
			Height:    evt.Height,
			Offset:    evt.Offset,
			Peer:      evt.Peer,
		})
	case events.BatchSyncRequestReceivedEventType:
		evt := ev.(*events.BatchSyncRequestReceivedEvent)
		txis := op.OgCore.LoadTxis(evt.Hashes, op.Config.MaxTxCountInResponse)
		op.EventBus.Route(&events.TxsFetchedForResponseEvent{
			Txs:       txis,
			RequestId: evt.RequestId,
			Peer:      evt.Peer,
		})
	case events.TxsFetchedForResponseEventType:
		evt := ev.(*events.TxsFetchedForResponseEvent)
		resources := make([]message.MessageContentResource, len(evt.Txs))
		for i, txi := range evt.Txs {
			resources[i] = op.AdaptTxiToResource(txi)
		}

		op.PeerOutgoing.Unicast(&message.OgMessageSyncResponse{
			RequestId: evt.RequestId,
			Height:    evt.Height,
			Offset:    evt.Offset,
			Resources: resources,
		}, evt.Peer)
	case events.NewTxLocallyGeneratedEventType:
		// in the future, do buffer
		evt := ev.(*events.NewTxLocallyGeneratedEvent)
		tx := evt.Tx

		if evt.RequireBroadcast {
			op.PeerOutgoing.Broadcast(&message.OgMessageNewResource{
				Resources: []message.MessageContentResource{
					op.AdaptTxiToResource(tx),
				},
			})
		}
	default:
		op.Logger.WithField("type", ev.GetEventType()).Warn("event type not supported")
	}
}

func (op *OgPartner) HandlerDescription(ev eventbus.EventType) string {
	switch ev {
	case events.HeightSyncRequestReceivedEventType:
		return "LoadHeightAndSendBack"
	case events.BatchSyncRequestReceivedEventType:
		return "LoadTxsByHashAndSendBack"
	case events.TxsFetchedForResponseEventType:
		return "SendingTxsFetchedResponse"
	case events.NewTxLocallyGeneratedEventType:
		return "BroadcastNewlyGeneratedTx"
	default:
		return "N/A"
	}
}

func (op *OgPartner) Name() string {
	return "OgPartner"
}

func (op *OgPartner) InitDefault() {
	op.quitWg = sync.WaitGroup{}
	op.quit = make(chan bool)
}

func (op *OgPartner) Start() {
	if op.quit == nil {
		panic("not initialized.")
	}
	go op.loop()
	op.Logger.Info("OgPartner Started")

}

func (op *OgPartner) Stop() {
	op.quit <- true
	op.quitWg.Wait()
	op.Logger.Debug("OgPartner Stopped")
}

func (op *OgPartner) loop() {
	for {
		select {
		case <-op.quit:
			op.quitWg.Done()
			op.Logger.Debug("OgPartner quit")
			return
		case msgEvent := <-op.PeerIncoming.GetPipeOut():
			op.HandleOgMessage(msgEvent)
		}
	}
}

func (op *OgPartner) HandleOgMessage(msgEvent *communication.OgMessageEvent) {
	op.Logger.WithField("msg", msgEvent).Trace("received an OG message")
	switch msgEvent.Message.GetType() {
	//case message.OgMessageTypeStatus:
	//	op.HandleMessageStatus(msgEvent)
	case message.OgMessageTypePing:
		op.HandleMessagePing(msgEvent)
	case message.OgMessageTypePong:
		op.HandleMessagePong(msgEvent)
	case message.OgMessageTypeQueryStatusRequest:
		op.HandleMessageQueryStatusRequest(msgEvent)
	case message.OgMessageTypeQueryStatusResponse:
		op.HandleMessageQueryStatusResponse(msgEvent)
	case message.OgMessageTypeNewResource:
		op.HandleMessageNewResource(msgEvent)
	case message.OgMessageTypeHeightSyncRequest:
		op.HandleMessageHeightSyncRequest(msgEvent)
	case message.OgMessageTypeBatchSyncRequest:
		op.HandleMessageBatchSyncRequest(msgEvent)
	case message.OgMessageTypeSyncResponse:
		op.HandleMessageSyncResponse(msgEvent)
	default:
		op.Logger.WithField("msg", msgEvent.Message).Warn("unsupported og message type")
	}
}

func (op *OgPartner) FireEvent(event eventbus.Event) {
	if op.EventBus != nil {
		op.EventBus.Route(event)
	}
}

func (op *OgPartner) HandleMessagePing(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	op.Logger.Debugf("received ping from %d. Respond you a pong.", source.Id)
	op.FireEvent(&events.PingReceivedEvent{})
	op.PeerOutgoing.Unicast(&message.OgMessagePong{}, source)

}

func (op *OgPartner) HandleMessagePong(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	op.Logger.Debugf("received pong from %d.", source.Id)
	op.FireEvent(&events.PongReceivedEvent{})
	op.PeerOutgoing.Unicast(&message.OgMessagePing{}, source)
}

func (op *OgPartner) HandleMessageQueryStatusRequest(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	op.Logger.Debugf("received QueryStatusRequest from %d.", source.Id)
	status := op.StatusProvider.GetCurrentOgStatus()
	op.PeerOutgoing.Unicast(&message.OgMessageQueryStatusResponse{
		ProtocolVersion: status.ProtocolVersion,
		NetworkId:       status.NetworkId,
		CurrentBlock:    status.CurrentBlock,
		GenesisBlock:    status.GenesisBlock,
		CurrentHeight:   status.CurrentHeight,
	}, source)
}

func (op *OgPartner) HandleMessageQueryStatusResponse(msgEvent *communication.OgMessageEvent) {
	// must be a response message
	resp, ok := msgEvent.Message.(*message.OgMessageQueryStatusResponse)
	if !ok {
		op.Logger.Warn("bad format: OgMessageQueryStatusResponse")
		return
	}
	source := msgEvent.Peer
	op.Logger.Debugf("received QueryStatusRequest from %d.", source.Id)
	statusData := model.OgStatusData{
		ProtocolVersion: resp.ProtocolVersion,
		NetworkId:       resp.NetworkId,
		CurrentBlock:    resp.CurrentBlock,
		GenesisBlock:    resp.GenesisBlock,
		CurrentHeight:   resp.CurrentHeight,
	}
	// this is ogcore inner process so do not involve event handler
	op.OgCore.HandleStatusData(statusData)
	op.FireEvent(&events.QueryStatusResponseReceivedEvent{
		StatusData: statusData,
	})
}

func (op *OgPartner) populateResource(resource message.MessageContentResource) {
	switch resource.ResourceType {
	case message.ResourceTypeTx:
		msgTx := &message.MessageContentTx{}
		err := msgTx.FromBytes(resource.ResourceContent)
		if err != nil {
			op.Logger.Warn("bad format: MessageContentTx")
			return
		}
		tx := &types.Tx{
			Hash:         msgTx.Hash,
			ParentsHash:  msgTx.ParentsHash,
			MineNonce:    msgTx.MineNonce,
			AccountNonce: msgTx.AccountNonce,
			From:         msgTx.From,
			To:           msgTx.To,
			Value:        msgTx.Value,
			TokenId:      msgTx.TokenId,
			Data:         msgTx.Data,
			PublicKey:    crypto.PublicKeyFromRawBytes(msgTx.PublicKey),
			Signature:    crypto.SignatureFromRawBytes(msgTx.Signature),
		}
		// og knows first
		//op.OgCore.HandleNewTx(tx)
		op.EventBus.Route(&events.TxReceivedEvent{Tx: tx})
	case message.ResourceTypeSequencer:
		msgSeq := &message.MessageContentSequencer{}
		err := msgSeq.FromBytes(resource.ResourceContent)
		if err != nil {
			op.Logger.Warn("bad format: MessageContentSequencer")
			return
		}
		seq := &types.Sequencer{
			Hash:         msgSeq.Hash,
			ParentsHash:  msgSeq.ParentsHash,
			Height:       msgSeq.Height,
			MineNonce:    msgSeq.MineNonce,
			AccountNonce: msgSeq.AccountNonce,
			Issuer:       msgSeq.Issuer,
			Signature:    msgSeq.Signature,
			PublicKey:    msgSeq.PublicKey,
			StateRoot:    msgSeq.StateRoot,
		}
		// og knows first
		//op.OgCore.HandleNewSequencer(seq)
		op.EventBus.Route(&events.SequencerReceivedEvent{Sequencer: seq})
		//case message.ResourceTypeArchive:
		//case message.ResourceTypeAction:
	}
}

func (op *OgPartner) HandleMessageNewResource(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageNewResource)
	if !ok {
		op.Logger.Warn("bad format: OgMessageNewResource")
		return
	}
	// decode all resources and announce it to the receivers.
	for _, resource := range msg.Resources {
		op.Logger.WithField("resource", resource.String()).Infof("received resource passively")
		op.populateResource(resource)
	}
}

func (op *OgPartner) SendMessagePing(peer *communication.OgPeer) {
	op.PeerOutgoing.Unicast(&message.OgMessagePing{}, peer)
}

func (op *OgPartner) SendMessageQueryStatusRequest(peer *communication.OgPeer) {
	op.PeerOutgoing.Unicast(&message.OgMessageQueryStatusRequest{}, peer)
}

func (op *OgPartner) SendMessageHeightSyncRequest(peer *communication.OgPeer) {
	op.PeerOutgoing.Unicast(&message.OgMessageHeightSyncRequest{
		Height:    1,
		Offset:    0,
		RequestId: 2,
	}, peer)
}

func (op *OgPartner) HandleMessageBatchSyncRequest(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageBatchSyncRequest)
	if !ok {
		op.Logger.Warn("bad format: HandleMessageBatchSyncRequest")
		return
	}
	op.EventBus.Route(&events.BatchSyncRequestReceivedEvent{
		Hashes:    msg.Hashes,
		RequestId: msg.RequestId,
		Peer:      msgEvent.Peer,
	})
}

func (op *OgPartner) HandleMessageHeightSyncRequest(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageHeightSyncRequest)
	if !ok {
		op.Logger.Warn("bad format: HandleMessageHeightSyncRequest")
		return
	}
	op.EventBus.Route(&events.HeightSyncRequestReceivedEvent{
		Height:    msg.Height,
		Offset:    msg.Offset,
		RequestId: msg.RequestId,
		Peer:      msgEvent.Peer,
	})
}

func (op *OgPartner) HandleMessageSyncResponse(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageSyncResponse)
	if !ok {
		op.Logger.Warn("bad format: HandleMessageSyncResponse")
		return
	}

	for _, resource := range msg.Resources {
		op.Logger.WithField("resource", resource.String()).Infof("received resource actively")
		op.populateResource(resource)
	}
}

type OgProcessorConfig struct {
	MaxTxCountInResponse int
}
