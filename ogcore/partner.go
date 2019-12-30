package ogcore

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/message"
	"github.com/annchain/OG/ogcore/model"
	"github.com/sirupsen/logrus"
	"sync"
)

type OgPartner struct {
	Config       OgProcessorConfig
	PeerOutgoing communication.OgPeerCommunicatorOutgoing
	PeerIncoming communication.OgPeerCommunicatorIncoming
	EventBus     EventBus

	// og protocols
	StatusProvider OgStatusProvider
	OgCore         *OgCore

	// partner should connect the backend service by announce events
	quit   chan bool
	quitWg sync.WaitGroup
}

func (a *OgPartner) AdaptTxiToResource(txi types.Txi) message.MessageContentResource {
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

func (a *OgPartner) HandleEvent(ev eventbus.Event) {
	// handle sending events
	switch ev.GetEventType() {
	case events.TxsFetchedForResponseEventType:
		evt := ev.(*events.TxsFetchedForResponseEvent)
		resources := make([]message.MessageContentResource, len(evt.Txs))
		for i, txi := range evt.Txs {
			resources[i] = a.AdaptTxiToResource(txi)
		}

		a.PeerOutgoing.Unicast(&message.OgMessageSyncResponse{
			RequestId: evt.RequestId,
			Height:    evt.Height,
			Offset:    evt.Offset,
			Resources: resources,
		}, evt.Peer)
	case events.NewTxLocallyGeneratedEventType:
		// in the future, do buffer
		evt := ev.(*events.NewTxLocallyGeneratedEvent)
		tx := evt.Tx

		a.PeerOutgoing.Broadcast(&message.OgMessageNewResource{
			Resources: []message.MessageContentResource{
				a.AdaptTxiToResource(tx),
			},
		})
	default:
		logrus.WithField("type", ev.GetEventType()).Warn("event type not supported")
	}
}

func (a *OgPartner) Name() string {
	return "OgPartner"
}

func (a *OgPartner) InitDefault() {
	a.quitWg = sync.WaitGroup{}
	a.quit = make(chan bool)
}

func (o *OgPartner) Start() {
	go o.loop()
	logrus.Info("OgPartner Started")

}

func (ap *OgPartner) Stop() {
	ap.quit <- true
	ap.quitWg.Wait()
	logrus.Debug("OgPartner Stopped")
}

func (o *OgPartner) loop() {
	for {
		select {
		case <-o.quit:
			o.quitWg.Done()
			logrus.Debug("OgPartner quit")
			return
		case msgEvent := <-o.PeerIncoming.GetPipeOut():
			o.HandleOgMessage(msgEvent)
		}
	}
}

func (o *OgPartner) HandleOgMessage(msgEvent *communication.OgMessageEvent) {

	switch msgEvent.Message.GetType() {
	//case message.OgMessageTypeStatus:
	//	o.HandleMessageStatus(msgEvent)
	case message.OgMessageTypePing:
		o.HandleMessagePing(msgEvent)
	case message.OgMessageTypePong:
		o.HandleMessagePong(msgEvent)
	case message.OgMessageTypeQueryStatusRequest:
		o.HandleMessageQueryStatusRequest(msgEvent)
	case message.OgMessageTypeQueryStatusResponse:
		o.HandleMessageQueryStatusResponse(msgEvent)
	case message.OgMessageTypeNewResource:
		o.HandleMessageNewResource(msgEvent)
	case message.OgMessageTypeHeightSyncRequest:
		o.HandleMessageHeightSyncRequest(msgEvent)
	case message.OgMessageTypeSyncResponse:
		o.HandleMessageTypeSyncResponse(msgEvent)
	default:
		logrus.WithField("msg", msgEvent.Message).Warn("unsupported og message type")
	}
}

func (o *OgPartner) FireEvent(event eventbus.Event) {
	if o.EventBus != nil {
		o.EventBus.Route(event)
	}
}

func (o *OgPartner) HandleMessagePing(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received ping from %d. Respond you a pong.", source.Id)
	o.FireEvent(&events.PingReceivedEvent{})
	o.PeerOutgoing.Unicast(&message.OgMessagePong{}, source)

}

func (o *OgPartner) HandleMessagePong(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received pong from %d.", source.Id)
	o.FireEvent(&events.PongReceivedEvent{})
	o.PeerOutgoing.Unicast(&message.OgMessagePing{}, source)
}

func (o *OgPartner) HandleMessageQueryStatusRequest(msgEvent *communication.OgMessageEvent) {
	source := msgEvent.Peer
	logrus.Debugf("received QueryStatusRequest from %d.", source.Id)
	status := o.StatusProvider.GetCurrentOgStatus()
	o.PeerOutgoing.Unicast(&message.OgMessageQueryStatusResponse{
		ProtocolVersion: status.ProtocolVersion,
		NetworkId:       status.NetworkId,
		CurrentBlock:    status.CurrentBlock,
		GenesisBlock:    status.GenesisBlock,
		CurrentHeight:   status.CurrentHeight,
	}, source)
}

func (o *OgPartner) HandleMessageQueryStatusResponse(msgEvent *communication.OgMessageEvent) {
	// must be a response message
	resp, ok := msgEvent.Message.(*message.OgMessageQueryStatusResponse)
	if !ok {
		logrus.Warn("bad format: OgMessageQueryStatusResponse")
		return
	}
	source := msgEvent.Peer
	logrus.Debugf("received QueryStatusRequest from %d.", source.Id)
	statusData := model.OgStatusData{
		ProtocolVersion: resp.ProtocolVersion,
		NetworkId:       resp.NetworkId,
		CurrentBlock:    resp.CurrentBlock,
		GenesisBlock:    resp.GenesisBlock,
		CurrentHeight:   resp.CurrentHeight,
	}
	// this is ogcore inner process so do not involve event handler
	o.OgCore.HandleStatusData(statusData)
	o.FireEvent(&events.QueryStatusResponseReceivedEvent{
		StatusData: statusData,
	})
}

func (o *OgPartner) HandleMessageNewResource(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageNewResource)
	if !ok {
		logrus.Warn("bad format: OgMessageNewResource")
		return
	}
	// decode all resources and announce it to the receivers.
	for _, resource := range msg.Resources {
		logrus.WithField("resource", resource.String()).Infof("Received resource")
		switch resource.ResourceType {
		case message.ResourceTypeTx:
			msgTx := &message.MessageContentTx{}
			err := msgTx.FromBytes(resource.ResourceContent)
			if err != nil {
				logrus.Warn("bad format: MessageContentTx")
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
			o.OgCore.HandleNewTx(tx)
			o.EventBus.Route(&events.TxReceivedEvent{Tx: tx})
		case message.ResourceTypeSequencer:
			msgSeq := &message.MessageContentSequencer{}
			err := msgSeq.FromBytes(resource.ResourceContent)
			if err != nil {
				logrus.Warn("bad format: MessageContentSequencer")
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
			o.OgCore.HandleNewSequencer(seq)
			o.EventBus.Route(&events.SequencerReceivedEvent{Sequencer: seq})
			//case message.ResourceTypeArchive:
			//case message.ResourceTypeAction:
		}
	}
}

func (o *OgPartner) SendMessagePing(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessagePing{}, peer)
}

func (o *OgPartner) SendMessageQueryStatusRequest(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessageQueryStatusRequest{}, peer)
}

func (o *OgPartner) SendMessageHeightSyncRequest(peer communication.OgPeer) {
	o.PeerOutgoing.Unicast(&message.OgMessageHeightSyncRequest{
		Height:    1,
		Offset:    0,
		RequestId: 2,
	}, peer)
}

func (a *OgPartner) HandleMessageBatchSyncRequest(msgEvent *communication.OgMessageEvent) {

}

func (a *OgPartner) HandleMessageHeightSyncRequest(msgEvent *communication.OgMessageEvent) {
	msg, ok := msgEvent.Message.(*message.OgMessageHeightSyncRequest)
	if !ok {
		logrus.Warn("bad format: HandleMessageHeightSyncRequest")
		return
	}
	a.EventBus.Route(&events.HeightSyncRequestReceivedEvent{
		Height:    msg.Height,
		Offset:    msg.Offset,
		RequestId: msg.RequestId,
	})
}

func (a *OgPartner) HandleMessageTypeSyncResponse(event *communication.OgMessageEvent) {

}

type OgProcessorConfig struct {
}
