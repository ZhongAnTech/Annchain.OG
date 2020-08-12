package og

//func (o *OgEngine) handleHeightRequest(letter *transport_interface.IncomingLetter) {
//	// report my height info
//	resp := message.OgMessageHeightResponse{
//		Height: o.CurrentHeight(),
//	}
//	o.notifyNewOutgoingMessage(&transport_interface.OutgoingLetter{
//		ExceptMyself:   true,
//		Msg:            &resp,
//		SendType:       transport_interface.SendTypeUnicast,
//		CloseAfterSent: false,
//		EndReceivers:   []string{letter.From},
//	})
//}
//
//func (o *OgEngine) handleHeightResponse(letter *transport_interface.IncomingLetter) {
//	// if we get a pong with close flag, the target peer is dropping us.
//	m := &message.OgMessageHeightResponse{}
//	err := m.FromBytes(letter.Msg.ContentBytes)
//	if err != nil {
//		logrus.WithField("type", m.GetType()).WithError(err).Warn("bad message")
//	}
//	if m.Height <= o.CurrentHeight() {
//		// already known that
//		logrus.WithField("theirHeight", m.Height).
//			WithField("myHeight", o.CurrentHeight()).
//			WithField("from", letter.From).Debug("detected a new height but is not higher than mine")
//		// still need to register this height in the ogsyncer
//		//return
//	}
//	// found a height that is higher that ours. Announce a new height received event
//	logrus.WithField("height", m.Height).WithField("from", letter.From).Debug("detected a new height")
//	o.notifyNewHeightDetected(&og_interface.NewHeightDetectedEvent{
//		Height: m.Height,
//		PeerId: letter.From,
//	})
//
//}

//func (o *OgEngine) handlePeerJoined(event *og_interface.PeerJoinedEvent) {
//	// detect height
//	m := &message.OgMessageHeightRequest{}
//	o.notifyNewOutgoingMessage(&transport_interface.OutgoingLetter{
//		ExceptMyself:   true,
//		Msg:            m,
//		SendType:       transport_interface.SendTypeUnicast,
//		CloseAfterSent: false,
//		EndReceivers:   []string{event.PeerId},
//	})
//}
