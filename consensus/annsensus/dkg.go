package annsensus

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/bls"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/tbls"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Dkg struct {
	ann *AnnSensus

	dkgOn   bool
	pk      []byte
	partner *Partner

	gossipStartCh  chan struct{}
	gossipStopChan chan struct{}
	gossipReqCh    chan *types.MessageConsensusDkgDeal
	gossipRespCh   chan *types.MessageConsensusDkgDealResponse
	mu             sync.RWMutex
}

func newDkg(ann *AnnSensus, dkgOn bool, numParts, threshold int) *Dkg {
	p := NewPartner(bn256.NewSuiteG2())
	p.NbParticipants = numParts
	p.Threshold = threshold
	p.PartPubs = []kyber.Point{}

	d := &Dkg{}
	d.ann = ann
	d.dkgOn = dkgOn
	d.partner = p
	d.gossipStartCh = make(chan struct{})
	d.gossipReqCh = make(chan *types.MessageConsensusDkgDeal, 100)
	d.gossipRespCh = make(chan *types.MessageConsensusDkgDealResponse, 100)
	d.gossipStopChan = make(chan struct{})
	d.dkgOn = dkgOn
	if d.dkgOn {
		d.GenerateDkg() //todo fix later
	}

	return d
}

func (d *Dkg) Reset() {
	p := NewPartner(bn256.NewSuiteG2())
	p.NbParticipants = d.partner.NbParticipants
	p.Threshold = d.partner.Threshold
	p.PartPubs = []kyber.Point{}

	d.partner = p
	d.pk = nil
}

func (d *Dkg) start() {
	//TODO
	log.Info("dkg start")
	go d.gossiploop()
}

func (d *Dkg) stop() {
	d.gossipStopChan <- struct{}{}
	log.Info("dkg stop")
}

func (d *Dkg) GenerateDkg() {
	sec, pub := genPartnerPair(d.partner)
	d.partner.MyPartSec = sec
	//d.partner.PartPubs = []kyber.Point{pub}??
	pk, _ := pub.MarshalBinary()
	d.pk = pk
}

func (d *Dkg) PublicKey() []byte {
	return d.pk
}

func (d *Dkg) StartGossip() {
	d.gossipStartCh <- struct{}{}
}

func (d *Dkg) loadCampaigns(camps []*types.Campaign) {
	for _, camp := range camps {
		d.partner.PartPubs = append(d.partner.PartPubs, camp.GetDkgPublicKey())
		d.partner.addressIndex[camp.Sender()] = len(d.partner.PartPubs) - 1
	}
}

func genPartnerPair(p *Partner) (kyber.Scalar, kyber.Point) {
	sc := p.Suite.Scalar().Pick(p.Suite.RandomStream())
	return sc, p.Suite.Point().Mul(sc, nil)
}

type DealsMap map[int]*dkg.Deal

func (t DealsMap) TerminateString() string {
	if t == nil || len(t) == 0 {
		return ""
	}
	var str []string
	var keys []int
	for k := range t {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		if v, ok := t[k]; ok {
			str = append(str, fmt.Sprintf("key-%d-val-%s", k, v.TerminateString()))
		}
	}
	return strings.Join(str, ",")
}

func (d *Dkg) getDeals() (DealsMap, error) {
	return d.partner.Dkger.Deals()
}

func (d *Dkg) gossiploop() {
	for {
		select {
		case <-d.gossipStartCh:
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}

			log := log.WithField("me ", d.ann.id)
			err := d.partner.GenerateDKGer()
			if err != nil {
				log.WithError(err).Error("gen dkger fail")
				continue
			}
			deals, err := d.getDeals()
			if err != nil {
				log.WithError(err).Error("generate dkg deal error")
				continue
			}
			//log.WithField("len deals", len(deals)).WithField("len partner",
			//	len(d.partner.PartPubs)).WithField("deals", deals.TerminateString()).Trace("got deals")
			for i, deal := range deals {
				data, _ := deal.MarshalMsg(nil)
				msg := &types.MessageConsensusDkgDeal{
					Data: data,
					Id:   og.MsgCounter.Get(),
				}
				addr := d.GetPartnerAddressByIndex(i)
				if addr == nil {
					panic("address not found")
				}
				if *addr == d.ann.MyPrivKey.PublicKey().Address() {
					//this is for me , skip myself
					log.WithField("i ", i).WithField("to ", addr.TerminalString()).WithField("deal",
						deal.TerminateString()).Error("send dkg deal to myself")
					panic("send dkg deal to myself")
					//d.gossipReqCh <- msg
					//continue
				}
				cp := d.ann.GetCandidate(*addr)
				if cp == nil {
					panic("campaign not found")
				}
				s := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
				msg.Sinature = s.Sign(*d.ann.MyPrivKey, msg.SignatureTargets()).Bytes
				msg.PublicKey = d.ann.MyPrivKey.PublicKey().Bytes
				pk := crypto.PublicKeyFromBytes(crypto.CryptoTypeSecp256k1, cp.PublicKey)
				log.WithField("i ", i).WithField("to ", addr.TerminalString()).WithField("deal",
					deal.TerminateString()).Debug("send dkg deal to")
				d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDeal, msg, &pk)
			}

		case request := <-d.gossipReqCh:
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}

			log := log.WithField("me ", d.ann.id)
			time.Sleep(10 * time.Millisecond) //this is for test
			var deal dkg.Deal
			_, err := deal.UnmarshalMsg(request.Data)
			if err != nil {
				log.Warn("unmarshal failed failed")
			}
			var cp *types.Campaign
			for _, v := range d.ann.Candidates() {
				if bytes.Equal(v.PublicKey, request.PublicKey) {
					cp = v
					break
				}
			}
			if cp == nil {
				log.WithField("deal ", request).Warn("not found  dkg  partner for deal")
				continue
			}

			s := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
			addr := s.AddressFromPubKeyBytes(request.PublicKey)
			if d.ann.GetCandidate(addr) == nil {
				log.WithField("pk ", request.Id).WithField("deal ", request).Warn("not found  campaign for deal")
				continue
			}
			_, ok := d.partner.addressIndex[addr]
			if !ok {
				log.WithField("deal ", request).Warn("not found  dkg partner dress   for deal")
				continue
			}
			d.mu.RLock()
			responseDeal, err := d.partner.Dkger.ProcessDeal(&deal)
			d.mu.RUnlock()
			if err != nil {
				log.WithField("deal ", request).WithError(err).Warn("  partner process error")
				continue
			}
			respData, err := responseDeal.MarshalMsg(nil)
			if err != nil {
				log.WithField("deal ", request).WithError(err).Warn("  partner process error")
				continue
			}
			response := &types.MessageConsensusDkgDealResponse{
				Data: respData,
				//Id:   request.Id,
				Id: og.MsgCounter.Get(),
			}
			signer := crypto.NewSigner(d.ann.cryptoType)
			response.Sinature = signer.Sign(*d.ann.MyPrivKey, response.SignatureTargets()).Bytes
			response.PublicKey = d.ann.MyPrivKey.PublicKey().Bytes
			log.WithField("to request ", request).WithField("response ", response).Debug("will send response")
			//broadcast response to all partner
			go d.ann.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, response)
			//and sent to myself ? already processed inside dkg,skip myself

		case response := <-d.gossipRespCh:
			log := log.WithField("me ", d.ann.id)
			var resp dkg.Response
			_, err := resp.UnmarshalMsg(response.Data)
			if err != nil {
				log.WithError(err).Warn("verify signature failed")
				return
			}
			//log.WithField("resp ", response).Trace("got response")
			//broadcast  continue
			go d.ann.Hub.BroadcastMessage(og.MessageTypeConsensusDkgDealResponse, response)
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}
			s := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
			addr := s.AddressFromPubKeyBytes(response.PublicKey)
			if d.ann.GetCandidate(addr) == nil {
				log.WithField("pub ", response.PublicKey).Trace("resp ")
				log.WithField("id ", response.Id).WithField("deal ", response).Error("not found  Campaign  for deal response")
				continue
			}
			_, ok := d.partner.addressIndex[addr]
			if !ok {
				log.WithField("address ", addr.TerminalString()).WithField("deal ", response).Warn(
					"not found  dkg  partner address  for deal")
				continue
			}
			d.mu.RLock()
			just, err := d.partner.Dkger.ProcessResponse(&resp)
			if err != nil {
				log.WithField("req ", response).WithField("rsp ", resp).WithField("just ", just).WithError(err).Warn("ProcessResponse failed")
			} else {
				log.WithField("response ", resp).Trace("process response ok")
				d.partner.responseNumber++
			}
			if d.partner.responseNumber >= (d.partner.NbParticipants-1)*(d.partner.NbParticipants-1) {
				log.Info("got response done")
				jointPub, err := d.partner.RecoverPub()
				if err != nil {
					log.WithError(err).Warn("get recover pub key fail")
				}
				// send public key to changeTerm loop.
				// TODO
				// this channel may be changed later.
				//d.ann.dkgPkCh <- jointPub
				log.WithField("bls key ", jointPub).Info("joint pubkey ")
			}
			d.mu.RUnlock()
			log.WithField("response number", d.partner.responseNumber).Trace("dkg")

		case <-d.gossipStopChan:
			log := log.WithField("me ", d.ann.id)
			log.Info("got quit signal dkg gossip stopped")
			return
		}
	}
}

func (d *Dkg) GetPartnerAddressByIndex(i int) *types.Address {

	for k, v := range d.partner.addressIndex {
		if v == i {
			return &k
		}
	}
	return nil
}

type Partner struct {
	PartPubs              []kyber.Point
	MyPartSec             kyber.Scalar
	addressIndex          map[types.Address]int
	SecretKeyContribution map[types.Address]kyber.Scalar
	Suite                 *bn256.Suite
	Dkger                 *dkg.DistKeyGenerator
	Resps                 map[types.Address]*dkg.Response
	Threshold             int
	NbParticipants        int
	jointPubKey           kyber.Point
	responseNumber        int
	SigShares             [][]byte
}

func NewPartner(s *bn256.Suite) *Partner {
	return &Partner{
		Suite:                 s,
		addressIndex:          make(map[types.Address]int),
		SecretKeyContribution: make(map[types.Address]kyber.Scalar),
		Resps: make(map[types.Address]*dkg.Response),
	}
}

func (p *Partner) GenerateDKGer() error {
	// use all partPubs and my partSec to generate a dkg
	dkger, err := dkg.NewDistKeyGenerator(p.Suite, p.MyPartSec, p.PartPubs, p.Threshold)
	if err != nil {
		log.WithField("dkger ", dkger).WithError(err).Error("generate dkg error")
		return err
	}
	p.Dkger = dkger
	return nil
}

func (p *Partner) VerifyByPubPoly(msg []byte, sig []byte) (err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	if pubPoly.Commit() != dks.Public() {
		err = errors.New("PubPoly not aligned to dksPublic")
		return
	}

	err = bls.Verify(p.Suite, pubPoly.Commit(), msg, sig)
	log.Debugf(" pubPolyCommit [%s] dksPublic [%s] dksCommitments [%s]\n",
		pubPoly.Commit(), dks.Public(), dks.Commitments())
	return
}

func (p *Partner) VerifyByDksPublic(msg []byte, sig []byte) (err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	err = bls.Verify(p.Suite, dks.Public(), msg, sig)
	return
}

func (p *Partner) RecoverSig(msg []byte) (jointSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointSig, err = tbls.Recover(p.Suite, pubPoly, msg, p.SigShares, p.Threshold, p.NbParticipants)
	return
}

func (p *Partner) RecoverPub() (jointPubKey kyber.Point, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointPubKey = pubPoly.Commit()
	p.jointPubKey = jointPubKey
	return
}

func (p *Partner) Sig(msg []byte) (partSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	partSig, err = tbls.Sign(p.Suite, dks.PriShare(), msg)
	return
}
