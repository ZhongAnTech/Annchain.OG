package dkg

import (
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	dkger "github.com/annchain/kyber/v3/share/dkg/pedersen"
	vss "github.com/annchain/kyber/v3/share/vss/pedersen"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// DkgPartner is the parter in a DKG group built to discuss a pub/privkey
// It will receive DKG messages and update the status.
// It is the handler for maintaining the DkgContext.
// Campaign or term change is not part of DKGPartner. Do their job in their own module.
type DkgPartner struct {
	context                  *DkgContext
	peerCommunicatorIncoming DkgPeerCommunicatorIncoming
	peerCommunicatorOutgoing DkgPeerCommunicatorOutgoing
	dealReceivingCache       DisorderedCache // map[deal_sender_index]Deal
	gossipStartCh            chan bool

	otherPeers            []PeerInfo
	notified              bool
	DealResponseCache     map[int]*dkger.Response //my response for such deal should not be generated twice
	ResponseCache         map[string]bool         // duplicate response should not be processed twice.
	total                 int
	dkgGeneratedListeners []DkgGeneratedListener // joint pubkey is got

	quit   chan bool
	quitWg sync.WaitGroup
}

func (p *DkgPartner) Stop() {
	close(p.quit)
	p.quitWg.Wait()
}

// NewDkgPartner inits a dkg group. All public keys should be already generated
// The public keys are shared before the Dkg group can be formed.
// This may be done by publishing partPub to the blockchain
// termId is still needed to identify different Dkg groups
// allPeers needs to be sorted and globally order identical
func NewDkgPartner(suite *bn256.Suite, termId uint32, numParts, threshold int, allPeers []PartPub, me PartSec,
	dkgPeerCommunicatorIncoming DkgPeerCommunicatorIncoming,
	dkgPeerCommunicatorOutgoing DkgPeerCommunicatorOutgoing) (*DkgPartner, error) {
	// new dkg context
	c := NewDkgContext(suite, termId)
	c.NbParticipants = numParts
	c.Threshold = threshold
	c.PartPubs = allPeers
	c.Me = me

	// find Who am I
	myIndex := -1
	for i := 0; i < len(allPeers); i++ {
		if allPeers[i].Point.Equal(me.Point) {
			// That's me
			myIndex = i
			break
		}
	}
	if myIndex == -1 {
		panic("did not find myself")
	}
	c.MyIndex = uint32(myIndex)

	// setup partner
	d := &DkgPartner{
		context:                  c,
		peerCommunicatorIncoming: dkgPeerCommunicatorIncoming,
		peerCommunicatorOutgoing: dkgPeerCommunicatorOutgoing,
		dealReceivingCache:       make(DisorderedCache),
		gossipStartCh:            make(chan bool),
		otherPeers:               []PeerInfo{},
		notified:                 false,
		DealResponseCache:        make(map[int]*dkger.Response),
		ResponseCache:            make(map[string]bool),
		total:                    0,
		dkgGeneratedListeners:    []DkgGeneratedListener{},
		quit:                     make(chan bool),
		quitWg:                   sync.WaitGroup{},
	}

	// init all other peers so that I can do broadcast
	for i := 0; i < len(allPeers); i++ {
		if i == int(c.MyIndex) {
			continue
		}
		d.otherPeers = append(d.otherPeers, allPeers[i].Peer)
	}

	if err := c.GenerateDKGer(); err != nil {
		// cannot build dkg group using these pubkeys
		return nil, err
	}
	return d, nil

}

// GenPartnerPair generates a part private/public key for discussing with others.
func GenPartnerPair(suite *bn256.Suite) (kyber.Scalar, kyber.Point) {
	sc := suite.Scalar().Pick(suite.RandomStream())
	return sc, suite.Point().Mul(sc, nil)
}

func (p *DkgPartner) Start() {
	// start to gossipLoop and share the deals
	p.quitWg.Add(1)
	goroutine.New(p.gossipLoop)
}

func (p *DkgPartner) gossipLoop() {
	select {
	case <-p.gossipStartCh:
		logrus.Debug("dkg gossip started")
		break
	case <-p.quit:
		logrus.Debug("dkg gossip quit")
		return
	}
	pipeOutChannel := p.peerCommunicatorIncoming.GetPipeOut()
	// send the deals to all other partners
	go p.announceDeals()
	for {
		timer := time.NewTimer(time.Second * 10)
		select {
		case <-p.quit:
			logrus.Warn("dkg gossip quit")
			p.quitWg.Done()
			return
		case <-timer.C:
			logrus.WithField("IM", p.context.Me.Peer.Address.ShortString()).Warn("Blocked reading incoming dkg")
			//p.checkWaitingForWhat()
		case msg := <-pipeOutChannel:
			logrus.WithField("me", p.context.MyIndex).WithField("type", msg.Type.String()).Trace("received a message")
			p.handleMessage(msg)
		}
	}

}

// announceDeals sends deals to all other partners to build up a dkg group
func (p *DkgPartner) announceDeals() {
	// get all deals that needs to be sent to other partners
	deals, err := p.context.Dkger.Deals()
	if err != nil {
		logrus.WithError(err).Fatal("failed to generate dkg deals")
	}
	for i, deal := range deals {
		p.sendDealToPartner(i, deal)
	}

	// in case of package loss, keep resending
	//for {
	//	time.Sleep(time.Second * 60)
	//	logrus.Warn("RESEND")
	//	for i, deal := range deals {
	//		p.sendDealToPartner(i, deal)
	//	}
	//}
}

// sendDealToPartner unicast a deal message to some specific partner
func (p *DkgPartner) sendDealToPartner(id int, deal *dkger.Deal) {
	data, err := deal.MarshalMsg(nil)
	if err != nil {
		logrus.WithError(err).Fatal("cannot marshal dkg deal")
	}

	msg := MessageDkgDeal{
		DkgBasicInfo: DkgBasicInfo{
			TermId: p.context.SessionId,
		},
		Data: data,
	}
	logrus.WithField("from", deal.Index).WithField("to", id).
		Trace("unicasting deal message")
	p.peerCommunicatorOutgoing.Unicast(p.wrapMessage(DkgMessageTypeDeal, &msg),
		p.context.PartPubs[id].Peer)
	// after this, you are expecting a response from the target peers
}

func (p *DkgPartner) sendResponseToAllRestPartners(response *dkger.Response) {
	data, err := response.MarshalMsg(nil)
	if err != nil {
		// TODO: change it to warn maybe
		logrus.WithError(err).Fatal("cannot marshal dkg response")
		return
	}

	msg := MessageDkgDealResponse{
		DkgBasicInfo: DkgBasicInfo{
			TermId: p.context.SessionId,
		},
		Data: data,
	}
	logrus.WithField("me", p.context.MyIndex).WithField("from", response.Response.Index).Trace("broadcasting response message")
	p.peerCommunicatorOutgoing.Broadcast(p.wrapMessage(DkgMessageTypeDealResponse, &msg), p.otherPeers)
}

func (p *DkgPartner) wrapMessage(messageType DkgMessageType, signable Signable) DkgMessage {
	m := DkgMessage{
		Type:    messageType,
		Payload: signable,
	}
	return m
}

func (p *DkgPartner) handleMessage(message DkgMessage) {
	switch message.Type {
	case DkgMessageTypeDeal:
		switch message.Payload.(type) {
		case *MessageDkgDeal:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("dkg msg payload error")
		}
		msg := message.Payload.(*MessageDkgDeal)
		p.handleDealMessage(msg)
	case DkgMessageTypeDealResponse:
		switch message.Payload.(type) {
		case *MessageDkgDealResponse:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("dkg msg payload error")
		}
		msg := message.Payload.(*MessageDkgDealResponse)
		p.handleDealResponseMessage(msg)
	default:
		logrus.WithField("type", message.Type).Warn("unknown dkg message type")
	}
}

func (p *DkgPartner) handleDealMessage(msg *MessageDkgDeal) {
	deal, err := msg.GetDeal()
	if err != nil {
		logrus.Warn("failed to unmarshal dkg deal message")
		return
	}
	// verify if deal's sender has a unified index and pubkey to avoid fake messages.
	err = p.verifyDealSender(msg, deal)
	if err != nil {
		logrus.WithError(err).Warn("wrong sender for dkg deal")
		return
	}

	issuerIndex := deal.Index
	v, hasPreviousDiscussion := p.dealReceivingCache[issuerIndex]
	if !hasPreviousDiscussion {
		v = &DkgDiscussion{
			Deal:      nil,
			Responses: []*dkger.Response{},
		}
		logrus.WithField("me", p.context.MyIndex).WithField("from", deal.Index).Trace("no previous discussion found in cache. new deal.")
	} else {
		logrus.WithField("me", p.context.MyIndex).WithField("from", deal.Index).Trace("previous discussion found in cache.")
	}

	//resp, inDealResponseCache := p.DealResponseCache[hexutil.Encode(deal.Signature)]
	resp, inDealResponseCache := p.DealResponseCache[int(deal.Index)]
	// if the resp is already generated, do not generate for the second time since it will error out.
	if !inDealResponseCache {
		// generate response and send out
		//p.dumpDeal(deal)
		//p.writeMessage(fmt.Sprintf("Process Deal %d", deal.Index))
		resp, err := p.context.Dkger.ProcessDeal(deal)
		if err != nil {
			logrus.WithError(err).Warn("failed to process deal")
			return
		}
		if resp.Response.Status != vss.StatusApproval {
			logrus.Warn("received a deal for rejection")
		}
		// cache the deal
		discussion := v.(*DkgDiscussion)
		discussion.Deal = deal

		// update state
		p.dealReceivingCache[issuerIndex] = discussion

		// send response to all other partners except itself
		//p.writeMessage(fmt.Sprintf("Send Resp %d %d", resp.Index, resp.Response.Index))
		p.sendResponseToAllRestPartners(resp)
		// avoid response regeneration
		//p.DealResponseCache[hexutil.Encode(deal.Signature)] = resp
		p.DealResponseCache[int(deal.Index)] = resp

		// TODO： BUGGY
		if hasPreviousDiscussion && discussion.GetCurrentStage() >= StageDealReceived {
			// now deal is just coming. Process the previous deal Responses
			for _, response := range discussion.Responses {
				//p.writeMessage(fmt.Sprintf("Process cached Resp %d %d", response.Index, response.Response.Index))
				p.handleResponse(response)
			}
			// clear the discussion response list since we won't process it twice
			discussion.Responses = []*dkger.Response{}
		} else {
			//p.writeMessage(fmt.Sprintf("not a cached way: %d", discussion.Deal.Index))
		}
	} else {
		// RE-send response to all other partners except itself
		//p.writeMessage(fmt.Sprintf("Send Resp %d %d", resp.Index, resp.Response.Index))
		p.sendResponseToAllRestPartners(resp)
	}
}

func (p *DkgPartner) handleDealResponseMessage(msg *MessageDkgDealResponse) {
	resp, err := msg.GetResponse()
	if err != nil {
		logrus.Warn("failed to unmarshal dkg response message")
		return
	}
	//p.dumpDealResponseMessage(resp, "received")
	// verify if response's sender has a unified index and pubkey to avoid fake messages.
	err = p.verifyResponseSender(msg, resp)
	if err != nil {
		logrus.WithError(err).Warn("wrong sender for dkg response")
		return
	}
	// avoid duplication
	signatureKey := hexutil.Encode(resp.Response.Signature)
	_, responseDuplicated := p.ResponseCache[signatureKey]
	if responseDuplicated {
		logrus.WithField("me", p.context.MyIndex).
			WithField("from", resp.Response.Index).
			WithField("deal", resp.Index).Trace("duplicate response, drop")
		return
	} else {
		p.ResponseCache[signatureKey] = true
	}

	// check if the correspondant deal is in the cache
	// if not, hang on
	dealerIndex := resp.Index
	v, ok := p.dealReceivingCache[dealerIndex]
	if !ok {
		// deal from this sender has not been received. put the response to the cache
		v = &DkgDiscussion{
			Deal:      nil,
			Responses: []*dkger.Response{},
		}
	}
	// currently whatever deal is there, append the response to the cache.
	// in the future this may be removed once deal is received.
	discussion := v.(*DkgDiscussion)
	discussion.Responses = append(discussion.Responses, resp)
	// update state
	p.dealReceivingCache[dealerIndex] = discussion
	//verifierIndex := resp.Response.Index

	// if deal is already there, process this response
	if discussion.Deal != nil || dealerIndex == p.context.MyIndex {
		logrus.WithField("me", p.context.MyIndex).
			WithField("from", resp.Response.Index).
			WithField("deal", resp.Index).Trace("new resp is being processed")
		//p.writeMessage(fmt.Sprintf("Process Resp %d %d", resp.Index, resp.Response.Index))
		p.handleResponse(resp)
	} else {
		logrus.WithField("me", p.context.MyIndex).
			WithField("from", resp.Response.Index).
			WithField("deal", resp.Index).Trace("new resp is being cached")
		//p.writeMessage(fmt.Sprintf("Cached Resp %d %d", resp.Index, resp.Response.Index))
	}
}

func (p *DkgPartner) handleResponse(resp *dkger.Response) {
	//p.dumpDealResponseMessage(resp, "realin")
	justification, err := p.context.Dkger.ProcessResponse(resp)
	if err != nil {
		logrus.WithError(err).WithField("me", p.context.MyIndex).
			WithField("from", resp.Response.Index).
			WithField("deal", resp.Index).Warn("error on processing response")
		return
	}
	if justification != nil {
		logrus.Warn("justification not nil")
		// TODO: broadcast the justificaiton to the others to inform that this is a bad node
	}
	logrus.WithField("me", p.context.MyIndex).
		WithField("from", resp.Response.Index).
		WithField("deal", resp.Index).Trace("response is ok")
	if !p.notified && p.context.Dkger.ThresholdCertified() {
		_, err := p.context.RecoverPub()
		if err != nil {
			logrus.WithField("me", p.context.MyIndex).Warn("DKG has been generated but pubkey reccovery failed")
		} else {
			logrus.WithField("me", p.context.MyIndex).WithField("pk", p.context.JointPubKey.String()).Info("DKG has been generated")
			p.notifyListeners()
			//p.checkWaitingForWhat()
		}
	}
}

func (p *DkgPartner) verifyDealSender(deal *MessageDkgDeal, deal2 *dkger.Deal) error {
	return nil
}

func (p *DkgPartner) verifyResponseSender(response *MessageDkgDealResponse, deal *dkger.Response) error {
	return nil
}

// notifyListeners notifies listeners who has been registered for dkg generated events
func (p *DkgPartner) notifyListeners() {
	for _, listener := range p.dkgGeneratedListeners {
		ffchan.NewTimeoutSenderShort(listener.GetDkgGeneratedEventChannel(), true, "listener")
		//listener.GetDkgGeneratedEventChannel() <- true
	}
	p.notified = true
}

func (p *DkgPartner) RegisterDkgGeneratedListener(l DkgGeneratedListener) {
	p.dkgGeneratedListeners = append(p.dkgGeneratedListeners, l)
}

//func (p *DkgPartner) GetPeerCommunicatorOutgoing() DkgPeerCommunicatorOutgoing {
//	return p.peerCommunicatorOutgoing
//}
//
//func (p *DkgPartner) GetPeerCommunicatorIncoming() DkgPeerCommunicatorIncoming {
//	return p.peerCommunicatorIncoming
//}

//
//func (p *DkgPartner) checkWaitingForWhat() {
//
//	//if p.context.MyIndex != 0 {notified
//	//	return
//	//}
//	total := TestNodes
//	if !p.notified {
//		logrus.WithField("IM", p.context.MyIndex).
//			WithField("qual", p.context.Dkger.QUAL()).
//			Warn("not notified")
//	} else {
//		return
//	}
//	logrus.WithField("me", p.context.MyIndex).
//		WithField("qual", len(p.context.Dkger.QUAL())).
//		WithField("notified", p.notified).
//		Info("check waiting for what")
//
//	var dealers []int
//
//	for dealer, v := range p.dealReceivingCache {
//		dealers = append(dealers, int(dealer))
//		discussion := v.(*DkgDiscussion)
//		if dealer != p.context.MyIndex && len(discussion.Responses) != total-2 {
//			logrus.WithFields(logrus.Fields{
//				"IM":  p.context.MyIndex,
//				"len": len(discussion.Responses),
//			}).Warn("missing")
//			for _, response := range discussion.Responses {
//				logrus.WithField("IM", p.context.MyIndex).WithField("deal index", response.Index).
//					WithField("resp index", response.Response.Index).Warn("resp index")
//			}
//		} else if dealer == p.context.MyIndex && len(discussion.Responses) != total-1 {
//			logrus.WithFields(logrus.Fields{
//				"IM":  p.context.MyIndex,
//				"len": len(discussion.Responses),
//			}).Warn("missing")
//			for _, response := range discussion.Responses {
//				logrus.WithField("IM", p.context.MyIndex).WithField("deal index", response.Index).
//					WithField("resp index", response.Response.Index).Warn("resp index")
//			}
//		}
//	}
//	sort.Ints(dealers)
//	logrus.WithField("dealers", dealers).WithField("ok", len(dealers) == total).WithField("IM", p.context.MyIndex).Info("all deals")
//}
//
//func (p *DkgPartner) dumpDeal(deal *dkger.Deal) {
//	return
//	debugPath := "D:/tmp/debug"
//	file, err := os.OpenFile(
//		path.Join(debugPath, fmt.Sprintf("deal_%02d.txt", p.context.MyIndex)),
//		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer file.Close()
//	_, err = fmt.Fprintf(file, "%d\r\n", deal.Index)
//	if err != nil {
//		panic(err)
//	}
//}
//
//func (p *DkgPartner) dumpDealResponseMessage(response *dkger.Response, msg string) {
//	p.total += 1
//	if p.context.MyIndex == 0  && p.total % 20 == 0{
//		fmt.Println(p.total)
//	}
//	return
//	debugPath := "D:/tmp/debug"
//	file, err := os.OpenFile(
//		path.Join(debugPath, fmt.Sprintf("resp_%02d_%s.txt", p.context.MyIndex, msg)),
//		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer file.Close()
//	_, err = fmt.Fprintf(file, "%d %d\r\n", response.Index, response.Response.Index)
//	if err != nil {
//		panic(err)
//	}
//}
//
//func (p *DkgPartner) writeMessage(msg string) {
//	return
//	debugPath := "D:/tmp/debug"
//	file, err := os.OpenFile(
//		path.Join(debugPath, fmt.Sprintf("log_%d.txt", p.context.MyIndex)),
//		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//	if err != nil {
//		panic(err)
//	}
//	defer file.Close()
//	_, err = fmt.Fprintf(file, "%s\r\n", msg)
//	if err != nil {
//		panic(err)
//	}
//}
