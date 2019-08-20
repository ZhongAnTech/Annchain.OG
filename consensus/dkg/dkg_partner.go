package dkg

import (
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	dkger "github.com/annchain/kyber/v3/share/dkg/pedersen"
	"github.com/sirupsen/logrus"
	"time"
)

// DkgPartner is the parter in a DKG group built to discuss a pub/privkey
// It will receive DKG messages and update the status.
// It is the handler for maintaining the DkgContext.
// Campaign or term change is not part of DKGPartner. Do their job in their own module.
type DkgPartner struct {
	context            *DkgContext
	PeerCommunicator   DkgPeerCommunicator
	DealReceivingCache DisorderedCache
	gossipStartCh      chan bool
	quit               chan bool
}

// NewDkgPartner inits a dkg group. All public keys should be already generated
// The public keys are shared before the Dkg group can be formed.
// This may be done by publishing partPub to the blockchain
// termId is still needed to identify different Dkg groups
func NewDkgPartner(termId uint64, numParts, threshold int, allPeers []PartPub, me PartSec,
	myPartSec PartSec) (*DkgPartner, error) {
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
			TermId: p.context.TermId,
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
			TermId: p.context.TermId,
		},
		Id:   0,
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
		p.handleDeal(msg)
	case DkgMessageTypeDealResponse:
		switch message.Payload.(type) {
		case *MessageDkgDealResponse:
		default:
			logrus.WithField("message.Payload", message.Payload).Warn("dkg msg payload error")
		}
		msg := message.Payload.(*MessageDkgDealResponse)
		p.handleDealResponse(msg)
	default:
		logrus.WithField("type", message.Type).Warn("unknown dkg message type")
	}
}

func (p *DkgPartner) handleDeal(msg *MessageDkgDeal) {
	var deal dkger.Deal
	_, err := deal.UnmarshalMsg(msg.Data)
	if err != nil {
		logrus.Warn("failed to unmarshal dkg deal message")
		return
	}
	p.DealReceivingCache[]
}

func (p *DkgPartner) handleDealResponse(msg *MessageDkgDealResponse) {

}
