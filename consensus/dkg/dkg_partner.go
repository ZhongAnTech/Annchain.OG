package dkg

import (
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	dkger "github.com/annchain/kyber/v3/share/dkg/pedersen"
	vss "github.com/annchain/kyber/v3/share/vss/pedersen"
	"github.com/sirupsen/logrus"
	"time"
)

// DkgPartner is the parter in a DKG group built to discuss a pub/privkey
// It will receive DKG messages and update the status.
// It is the handler for maintaining the DkgContext.
// Campaign or term change is not part of DKGPartner. Do their job in their own module.
type DkgPartner struct {
	context                 *DkgContext
	PeerCommunicator        DkgPeerCommunicator
	DealReceivingCache      DisorderedCache // map[deal_sender_address]Deal
	gossipStartCh           chan bool
	quit                    chan bool
	OnDkgPublicKeyJointChan chan bool // joint pubkey is got
}

// NewDkgPartner inits a dkg group. All public keys should be already generated
// The public keys are shared before the Dkg group can be formed.
// This may be done by publishing partPub to the blockchain
// termId is still needed to identify different Dkg groups
// allPeers needs to be sorted and globally order identical
func NewDkgPartner(termId uint64, numParts, threshold int, allPeers []PartPub, me PartSec) (*DkgPartner, error) {
	c := NewDkgContext(bn256.NewSuiteG2(), termId)
	c.NbParticipants = numParts
	c.Threshold = threshold

	c.PartPubs = allPeers
	c.Me = me

	d := &DkgPartner{}
	d.context = c
	d.gossipStartCh = make(chan bool)
	d.quit = make(chan bool)
	d.DealReceivingCache = make(DisorderedCache)

	if err := c.GenerateDKGer(); err != nil {
		// cannot build dkg group using these pubkeys
		return nil, err
	}
	return d, nil

}

// GenPartnerPair generates a part private/public key for discussing with others.
func GenPartnerPair(p *DkgContext) (kyber.Scalar, kyber.Point) {
	sc := p.Suite.Scalar().Pick(p.Suite.RandomStream())
	return sc, p.Suite.Point().Mul(sc, nil)
}

func (p *DkgPartner) Start() {
	// start to gossipLoop and share the deals
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
	timer := time.NewTimer(time.Second * 7)
	incomingChannel := p.PeerCommunicator.GetIncomingChannel()

	for {
		// send the deals to all other partners
		p.announceDeals()
		select {
		case <-p.quit:
			logrus.Debug("dkg gossip quit")
			return
		case <-timer.C:
			logrus.WithField("IM", p.context.Me.Peer.Address.ShortString()).Debug("Blocked reading incoming dkg")
		case msg := <-incomingChannel:
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
		Id:   0,
		Data: data,
	}
	p.PeerCommunicator.Unicast(p.wrapMessage(DkgMessageTypeDeal, &msg),
		p.context.PartPubs[id].Peer)
	// after this, you are expecting a response from the target peers
}

func (p *DkgPartner) sendResponseToAllPartners(response *dkger.Response) {
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

	p.PeerCommunicator.Broadcast(p.wrapMessage(DkgMessageTypeDealResponse, &msg),
		PartPubs(p.context.PartPubs).Peers())
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
	v, ok := p.DealReceivingCache[issuerIndex]
	if !ok {
		v = &DkgDiscussion{
			Deal:      nil,
			Responses: []*dkger.Response{},
		}
	}

	// give deal response
	resp, err := p.context.Dkger.ProcessDeal(deal)
	if err != nil {
		logrus.WithError(err).Warn("failed to process deal")
	}
	if resp.Response.Status != vss.StatusApproval {
		logrus.Warn("received a deal for rejection")
	}

	// cache the deal
	discussion := v.(*DkgDiscussion)
	discussion.Deal = deal
	// update state
	p.DealReceivingCache[issuerIndex] = discussion

	// send response to all other partners except itself
	p.sendResponseToAllPartners(resp)
	if !ok && discussion.GetCurrentStage() >= StageDealReceived {
		// now deal is just coming. Process the previous deal Responses
		for _, response := range discussion.Responses {
			p.handleResponse(response)
		}
	}
	if p.context.Dkger.Certified() {
		logrus.WithField("pk", p.context.JointPubKey.String()).Warn("DKG has been generated")
	}
}

func (p *DkgPartner) handleDealResponseMessage(msg *MessageDkgDealResponse) {
	resp, err := msg.GetResponse()
	if err != nil {
		logrus.Warn("failed to unmarshal dkg response message")
		return
	}
	// verify if response's sender has a unified index and pubkey to avoid fake messages.
	err = p.verifyResponseSender(msg, resp)
	if err != nil {
		logrus.WithError(err).Warn("wrong sender for dkg response")
		return
	}
	// check if the correspondant deal is in the cache
	// if not, hang on
	dealerIndex := resp.Index
	v, ok := p.DealReceivingCache[dealerIndex]
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
	p.DealReceivingCache[dealerIndex] = discussion
	//verifierIndex := resp.Response.Index

	// if deal is already there, process this response
	if ok {
		p.handleResponse(resp)
	}
}

func (p *DkgPartner) handleResponse(resp *dkger.Response) {
	justification, err := p.context.Dkger.ProcessResponse(resp)
	if err != nil {
		logrus.WithError(err).Warn("error on processing response")
	}
	if justification != nil {
		logrus.Warn("justification not nil")
		// TODO: broadcast the justificaiton to the others to inform that this is a bad node
	}
}

func (p *DkgPartner) verifyDealSender(deal *MessageDkgDeal, deal2 *dkger.Deal) error {
	return nil
}

func (p *DkgPartner) verifyResponseSender(response *MessageDkgDealResponse, deal *dkger.Response) error {

	return nil

}
