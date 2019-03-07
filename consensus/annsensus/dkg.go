// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package annsensus

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"

	log "github.com/sirupsen/logrus"
)

type Dkg struct {
	ann *AnnSensus

	dkgOn             bool
	pk                []byte
	partner           *Partner
	gossipStartCh     chan struct{}
	gossipStopCh      chan struct{}
	gossipReqCh       chan *types.MessageConsensusDkgDeal
	gossipRespCh      chan *types.MessageConsensusDkgDealResponse
	gossipSigSetspCh  chan *types.MessageConsensusDkgSigSets
	dealCache         map[types.Address]*types.MessageConsensusDkgDeal
	dealResPonseCache map[types.Address][]*types.MessageConsensusDkgDealResponse
	dealSigSetsCache  map[types.Address]*types.MessageConsensusDkgSigSets
	respWaitingCache  map[uint32][]*types.MessageConsensusDkgDealResponse
	blsSigSets        map[types.Address]*types.SigSet
	ready             bool

	mu     sync.RWMutex
	signer crypto.Signer
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
	d.gossipSigSetspCh = make(chan *types.MessageConsensusDkgSigSets, 100)
	d.gossipStopCh = make(chan struct{})
	d.dkgOn = dkgOn
	if d.dkgOn {
		d.GenerateDkg() //todo fix later
	}
	d.dealCache = make(map[types.Address]*types.MessageConsensusDkgDeal)
	d.dealResPonseCache = make(map[types.Address][]*types.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*types.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[types.Address]*types.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[types.Address]*types.SigSet)
	d.ready = false
	d.signer = crypto.NewSigner(d.ann.cryptoType)
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
	d.gossipStopCh <- struct{}{}
	log.Info("dkg stop")
}

func (d *Dkg) GenerateDkg() {
	d.mu.Lock()
	defer d.mu.Unlock()

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
	//for _, camp := range camps {
	//	d.partner.PartPubs = append(d.partner.PartPubs, camp.GetDkgPublicKey())
	//	d.partner.addressIndex[camp.Sender()] = len(d.partner.PartPubs) - 1
	//}
}

func (d *Dkg) getDeals() (DealsMap, error) {
	return d.partner.Dkger.Deals()
}

func (d *Dkg) AddPartner(c *types.Campaign, annPriv *crypto.PrivateKey) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.partner.PartPubs = append(d.partner.PartPubs, c.GetDkgPublicKey())
	if bytes.Equal(c.PublicKey, annPriv.PublicKey().Bytes) {
		d.partner.Id = uint32(len(d.partner.PartPubs) - 1)
	}
	d.partner.addressIndex[c.Issuer] = len(d.partner.PartPubs) - 1

}

func (d *Dkg) GetBlsSigsets() []*types.SigSet {
	var sigset []*types.SigSet
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, sig := range d.blsSigSets {
		sigset = append(sigset, sig)
	}
	return sigset
}

func (d *Dkg) GenerateDKGer() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.partner.GenerateDKGer()
}

func (d *Dkg) ProcesssDeal(dd *dkg.Deal) (resp *dkg.Response, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.partner.Dkger.ProcessDeal(dd)
}

func (d *Dkg) CheckAddress(addr types.Address) bool {
	if d.ann.GetCandidate(addr) == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.partner.addressIndex[addr]
	return ok
}

func (d *Dkg) getUnhandled() ([]*types.MessageConsensusDkgDeal, []*types.MessageConsensusDkgDealResponse) {
	var unhandledDeal []*types.MessageConsensusDkgDeal
	var unhandledResponse []*types.MessageConsensusDkgDealResponse
	d.mu.RLock()
	defer d.mu.RUnlock()
	for i := range d.dealCache {
		unhandledDeal = append(unhandledDeal, d.dealCache[i])
	}
	for i := range d.dealResPonseCache {
		unhandledResponse = append(unhandledResponse, d.dealResPonseCache[i]...)
	}
	if len(d.dealCache) == 0 {
		d.dealCache = make(map[types.Address]*types.MessageConsensusDkgDeal)
	}
	if len(d.dealResPonseCache) == 0 {
		d.dealResPonseCache = make(map[types.Address][]*types.MessageConsensusDkgDealResponse)
	}
	return unhandledDeal, unhandledResponse
}

func (d *Dkg) sendUnhandledTochan(unhandledDeal []*types.MessageConsensusDkgDeal, unhandledResponse []*types.MessageConsensusDkgDealResponse) {
	if len(unhandledDeal) != 0 || len(unhandledResponse) != 0 {
		log.WithField("unhandledDeal deals ", len(unhandledDeal)).WithField(
			"unhandledResponse", unhandledResponse).Debug("will process")

		for _, v := range unhandledDeal {
			d.gossipReqCh <- v
		}
		for _, v := range unhandledResponse {
			d.gossipRespCh <- v
		}
		log.WithField("unhandledDeal deals ", len(unhandledDeal)).WithField(
			"unhandledResponse", len(unhandledResponse)).Debug("processed ")
	}
}

func (d *Dkg) unhandledSigSets() []*types.MessageConsensusDkgSigSets {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var sigs []*types.MessageConsensusDkgSigSets
	for _, v := range d.dealSigSetsCache {
		sigs = append(sigs, v)
	}
	d.dealSigSetsCache = make(map[types.Address]*types.MessageConsensusDkgSigSets)
	return sigs
}

func (d *Dkg) ProcessDeal(deal *dkg.Deal) (*dkg.Response, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	responseDeal, err := d.partner.Dkger.ProcessDeal(deal)
	if err == nil {
		d.partner.dealsIndex[deal.Index] = true
	}
	return responseDeal, err

}

//ProcessWaitingResponse
func (d *Dkg) ProcessWaitingResponse(deal *dkg.Deal) {
	var resps []*types.MessageConsensusDkgDealResponse
	d.mu.RLock()
	resps, _ = d.respWaitingCache[deal.Index]
	delete(d.respWaitingCache, deal.Index)
	d.mu.RUnlock()
	if len(resps) != 0 {
		log.WithField("for index ", deal.Index).WithField("resps ", resps).Debug("will process cached response")
		for _, r := range resps {
			d.gossipRespCh <- r
		}
		log.WithField("resps ", resps).Debug("processed cached response")
	}
}

func (d *Dkg) sendDealsToCorrespondingPartner(deals DealsMap) {
	//for generating deals, we use n partPubs, including our partPub. it generates  n-1 deals, excluding our own deal
	//skip myself
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
		}
		//not a candidate
		cp := d.ann.GetCandidate(*addr)
		if cp == nil {
			panic("campaign not found")
		}
		msg.Sinature = d.signer.Sign(*d.ann.MyPrivKey, msg.SignatureTargets()).Bytes
		msg.PublicKey = d.ann.MyPrivKey.PublicKey().Bytes
		pk := crypto.PublicKeyFromBytes(d.ann.cryptoType, cp.PublicKey)
		log.WithField("to ", addr.TerminalString()).WithField("deal",
			deal.TerminateString()).WithField("msg", msg).Debug("send dkg deal to")
		d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDeal, msg, &pk)
	}
	return
}

func (d *Dkg) gossiploop() {
	log := log.WithField("me", d.partner.Id)
	for {
		select {
		case <-d.gossipStartCh:

			if !d.dkgOn {
				//i am not a consensus partner
				log.Warn("why send to me")
				continue
			}
			err := d.GenerateDKGer()
			if err != nil {
				log.WithError(err).Error("gen dkger fail")
				continue
			}

			deals, err := d.getDeals()
			if err != nil {
				log.WithError(err).Error("generate dkg deal error")
				continue
			}
			unhandledDeal, unhandledResponse := d.getUnhandled()
			d.mu.RLock()
			//dkg is ready now, can process dkg msg
			d.ready = true
			d.mu.RUnlock()
			d.sendDealsToCorrespondingPartner(deals)
			//process unhandled(cached) dkg msg
			d.sendUnhandledTochan(unhandledDeal, unhandledResponse)

		case request := <-d.gossipReqCh:
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}

			addr := d.signer.AddressFromPubKeyBytes(request.PublicKey)
			if !d.CacheDealsIfNotReady(addr, request) {
				log.WithField("deal ", request).Debug("process later ,not ready yet")
				continue
			}
			var deal dkg.Deal
			_, err := deal.UnmarshalMsg(request.Data)
			if err != nil {
				log.WithError(err).Warn("unmarshal failed")
				continue
			}
			if !d.CheckAddress(addr) {
				log.WithField("deal ", request).Warn("not found  dkg partner address  or campaign  for deal")
				continue
			}
			responseDeal, err := d.ProcessDeal(&deal)
			if err != nil {
				log.WithField("deal ", request).WithError(err).Warn("deal process error")
				continue
			}
			respData, err := responseDeal.MarshalMsg(nil)
			if err != nil {
				log.WithField("deal ", request).WithError(err).Warn("deal process error")
				continue
			}

			response := types.MessageConsensusDkgDealResponse{
				Data: respData,
				//Id:   request.Id,
				Id: og.MsgCounter.Get(),
			}
			response.Sinature = d.signer.Sign(*d.ann.MyPrivKey, response.SignatureTargets()).Bytes
			response.PublicKey = d.ann.MyPrivKey.PublicKey().Bytes
			log.WithField("to request ", request).WithField("response ", &response).Debug("will send response")
			//broadcast response to all partner
			me := d.ann.MyPrivKey.PublicKey().Address()
			for k, v := range d.ann.Candidates() {
				if k == me {
					continue
				}
				msgCopy := response
				pk := crypto.PublicKeyFromBytes(d.ann.cryptoType, v.PublicKey)
				go d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgDealResponse, &msgCopy, &pk)
			}
			//and sent to myself ? already processed inside dkg,skip myself
			d.ProcessWaitingResponse(&deal)

		case response := <-d.gossipRespCh:

			var resp dkg.Response
			_, err := resp.UnmarshalMsg(response.Data)
			if err != nil {
				log.WithError(err).Warn("verify signature failed")
				continue
			}
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}
			addr := d.signer.AddressFromPubKeyBytes(response.PublicKey)
			if !d.CacheResponseIfNotReady(addr, response) {
				log.WithField("response ", resp).Debug("process later ,not ready yet")
				continue
			}

			if !d.CheckAddress(addr) {
				log.WithField("address ", addr.TerminalString()).WithField("resp ", response).WithField("deal ", response).Warn(
					"not found  dkg  partner or campaign msg for address  of this  deal")
				continue
			}
			if !d.CacheResponseIfDealNotReceived(&resp, response) {
				log.WithField("for index ", resp.Index).Debug("deal  not received yet")
				continue
			}

			just, err := d.ProcessResponse(&resp)
			if err != nil {
				log.WithField("req ", response).WithField("rsp ", resp).WithField("just ", just).WithError(err).Warn("ProcessResponse failed")
				continue
			}
			log.WithField("response ", resp).Trace("process response ok")
			d.mu.RLock()
			d.partner.responseNumber++
			d.mu.RUnlock()
			log.WithField("response number", d.partner.responseNumber).Trace("dkg")
			//will got  (n-1)*(n-1) response
			if d.partner.responseNumber < (d.partner.NbParticipants-1)*(d.partner.NbParticipants-1) {

				continue
			}
			log.Info("got response done")
			d.mu.RLock()
			jointPub, err := d.partner.RecoverPub()
			d.mu.RUnlock()
			if err != nil {
				log.WithError(err).Warn("get recover pub key fail")
				continue
			}
			// send public key to changeTerm loop.
			// TODO
			// this channel may be changed later.
			log.WithField("bls key ", jointPub).Info("joint pubkey ")
			//d.ann.dkgPkCh <- jointPub
			var msg types.MessageConsensusDkgSigSets
			msg.PkBls, _ = jointPub.MarshalBinary()
			msg.Sinature = d.signer.Sign(*d.ann.MyPrivKey, msg.SignatureTargets()).Bytes
			msg.PublicKey = d.ann.MyPrivKey.PublicKey().Bytes
			me := d.ann.MyPrivKey.PublicKey().Address()
			for k, v := range d.ann.Candidates() {
				if k == me {
					continue
				}
				msgCopy := msg
				pk := crypto.PublicKeyFromBytes(d.ann.cryptoType, v.PublicKey)
				go d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgSigSets, &msgCopy, &pk)
			}

			d.addSigsets(d.ann.MyPrivKey.PublicKey().Address(), &types.SigSet{PublicKey: msg.PublicKey, Signature: msg.Sinature})
			sigCaches := d.unhandledSigSets()
			for _, sigSets := range sigCaches {
				d.gossipSigSetspCh <- sigSets
			}

		case response := <-d.gossipSigSetspCh:

			pkBls, err := bn256.UnmarshalBinaryPointG2(response.PkBls)
			if err != nil {
				log.WithError(err).Warn("verify signature failed")
				continue
			}
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}
			s := crypto.NewSigner(d.ann.cryptoType)
			addr := s.AddressFromPubKeyBytes(response.PublicKey)
			if !d.CacheSigSetsIfNotReady(addr, response) {
				log.WithField("response ",
					response).Debug("process later sigsets ,not ready yet")
				continue
			}
			if !d.CheckAddress(addr) {
				log.WithField("address ", addr.TerminalString()).WithField("resp ", response).WithField("deal ", response).Warn(
					"not found  dkg  partner or campaign msg for address  of this  deal")
				continue
			}
			if !pkBls.Equal(d.partner.jointPubKey) {
				log.WithField("got pkbls ", pkBls).WithField("joint pk ", d.partner.jointPubKey).Warn("pk bls mismatch")
				continue
			}
			d.addSigsets(addr, &types.SigSet{PublicKey: response.PublicKey, Signature: response.Sinature})

			if len(d.blsSigSets) >= d.partner.NbParticipants {
				log.Info("got enough sig sets")
				d.ann.dkgPkCh <- d.partner.jointPubKey
			}

		case <-d.gossipStopCh:
			log.Info("got quit signal dkg gossip stopped")
			return
		}
	}
}

//CacheResponseIfDealNotReceived
func (d *Dkg) CacheResponseIfDealNotReceived(resp *dkg.Response, response *types.MessageConsensusDkgDealResponse) (ok bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if v := d.partner.dealsIndex[resp.Index]; !v && resp.Index != d.partner.Id {
		//if addr is my address , it  is not in the index list ,process them
		resps, _ := d.respWaitingCache[resp.Index]
		resps = append(resps, response)
		d.respWaitingCache[resp.Index] = resps
		log.WithField("cached resps ", resps).Debug("cached")
		return false
	}
	return true
}

//CacheResponseIfNotReady check whether dkg is ready, not ready  cache the dkg deal response
func (d *Dkg) CacheResponseIfNotReady(addr types.Address, response *types.MessageConsensusDkgDealResponse) (ready bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ready {
		dkgResps, _ := d.dealResPonseCache[addr]
		dkgResps = append(dkgResps, response)
		d.dealResPonseCache[addr] = dkgResps
	}
	return d.ready
}

//CacheDealsIfNotReady check whether dkg is ready, not ready  cache the dkg deal
func (d *Dkg) CacheDealsIfNotReady(addr types.Address, request *types.MessageConsensusDkgDeal) (ready bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ready {
		if oldDeal, ok := d.dealCache[addr]; ok {
			log.WithField("old ", oldDeal).WithField("new ", request).Warn("got duplicate deal")
		} else {
			d.dealCache[addr] = request
		}
	}
	return d.ready
}

//CacheSigSetsIfNotReady
func (d *Dkg) CacheSigSetsIfNotReady(addr types.Address, response *types.MessageConsensusDkgSigSets) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ready || d.partner.jointPubKey == nil {
		if _, ok := d.dealSigSetsCache[addr]; !ok {
			d.dealSigSetsCache[addr] = response
		}
		return false

	}
	return true
}

func (d *Dkg) addSigsets(addr types.Address, sig *types.SigSet) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.blsSigSets[addr] = sig
	return

}

func (d *Dkg) ProcessResponse(resp *dkg.Response) (just *dkg.Justification, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.processResponse(resp)
}

func (d *Dkg) processResponse(resp *dkg.Response) (just *dkg.Justification, err error) {
	return d.partner.Dkger.ProcessResponse(resp)
}

func (d *Dkg) GetPartnerAddressByIndex(i int) *types.Address {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for k, v := range d.partner.addressIndex {
		if v == i {
			return &k
		}
	}
	return nil
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
