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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"sort"
	"strings"
	"sync"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/vss/pedersen"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/bls"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

type Dkg struct {
	ann               *AnnSensus
	TermId            int
	dkgOn             bool
	myPublicKey       []byte
	partner           *DKGPartner
	formerPartner     *DKGPartner //used for case : partner reset , but bft is still generating sequencer
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
	isValidPartner    bool

	mu     sync.RWMutex
	signer crypto.Signer
}

func newDkg(ann *AnnSensus, dkgOn bool, numParts, threshold int) *Dkg {
	p := NewDKGPartner(bn256.NewSuiteG2())
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
		d.myPublicKey = d.partner.CandidatePublicKey
		d.partner.MyPartSec = d.partner.CandidatePartSec
	}
	d.dealCache = make(map[types.Address]*types.MessageConsensusDkgDeal)
	d.dealResPonseCache = make(map[types.Address][]*types.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*types.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[types.Address]*types.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[types.Address]*types.SigSet)
	d.ready = false
	d.isValidPartner = false
	d.signer = crypto.NewSigner(d.ann.cryptoType)
	return d
}

func (d *Dkg) Reset() {

	d.mu.RLock()
	defer d.mu.RUnlock()
	partSec := d.partner.CandidatePartSec
	pubKey := d.partner.CandidatePublicKey
	d.formerPartner = d.partner
	p := NewDKGPartner(bn256.NewSuiteG2())
	p.NbParticipants = d.partner.NbParticipants
	p.Threshold = d.partner.Threshold
	p.PartPubs = []kyber.Point{}
	p.MyPartSec = partSec
	d.dkgOn = false
	d.partner = p
	d.myPublicKey = pubKey
	d.TermId++
	d.dealCache = make(map[types.Address]*types.MessageConsensusDkgDeal)
	d.dealResPonseCache = make(map[types.Address][]*types.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*types.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[types.Address]*types.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[types.Address]*types.SigSet)
	d.ready = false
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

func (d *Dkg) GenerateDkg() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()

	sec, pub := genPartnerPair(d.partner)
	d.partner.CandidatePartSec = sec
	//d.partner.PartPubs = []kyber.Point{pub}??
	pk, _ := pub.MarshalBinary()
	d.partner.CandidatePublicKey = pk
	return pk
}

//PublicKey current pk
func (d *Dkg) PublicKey() []byte {
	return d.myPublicKey
}

func (d *Dkg) StartGossip() {
	d.gossipStartCh <- struct{}{}
}

type VrfSelection struct {
	addr   types.Address
	Vrf    hexutil.Bytes
	XORVRF hexutil.Bytes
	Id     int //for test
}

type VrfSelections []VrfSelection

func (a VrfSelections) Len() int      { return len(a) }
func (a VrfSelections) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a VrfSelections) Less(i, j int) bool {
	return hexutil.Encode(a[i].XORVRF) < hexutil.Encode(a[j].XORVRF)
}

func (v VrfSelection) String() string {
	return fmt.Sprintf("id-%d-a-%s-v-%s-xor-%s", v.Id, hexutil.Encode(v.addr.Bytes[:4]), hexutil.Encode(v.Vrf[:4]), hexutil.Encode(v.XORVRF[:4]))
}

// XOR takes two byte slices, XORs them together, returns the resulting slice.
func XOR(a, b []byte) []byte {
	c := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}
	return c
}

func (d *Dkg) SelectCandidates() {
	d.mu.RLock()
	defer d.mu.RUnlock()
	seq := d.ann.Idag.LatestSequencer()

	if len(d.ann.term.campaigns) == d.ann.NbParticipants {
		log.Debug("campaign number is equal to participant number ,all will be senator")
		var txs types.Txis
		for _, cp := range d.ann.term.campaigns {
			if bytes.Equal(cp.PublicKey, d.ann.MyAccount.PublicKey.Bytes) {
				d.isValidPartner = true
			}
			txs = append(txs, cp)
		}
		sort.Sort(txs)
		for _, tx := range txs {
			cp := tx.(*types.Campaign)
			d.ann.term.AddCandidate(cp)
			if d.isValidPartner {
				d.addPartner(cp)
				d.dkgOn = true
				d.GenerateDkg()
			}
		}
		return
	}
	if len(d.ann.term.campaigns) < d.ann.NbParticipants {
		panic("never come here , programmer error")
	}
	randomSeed := CalculateRandomSeed(seq.BlsJointSig)
	log.WithField("rand seed ", hexutil.Encode(randomSeed)).Debug("generated")
	var vrfSelections VrfSelections
	var j int
	d.ann.term.mu.RLock()
	for addr, cp := range d.ann.term.campaigns {
		vrfSelect := VrfSelection{
			addr:   addr,
			Vrf:    cp.Vrf.Vrf,
			XORVRF: XOR(cp.Vrf.Vrf, randomSeed),
			Id:     j,
		}
		j++
		vrfSelections = append(vrfSelections, vrfSelect)
	}
	d.ann.term.mu.RUnlock()
	log.Debugf("we have %d capmpaigns, select %d of them ", len(d.ann.term.campaigns), d.ann.NbParticipants)
	for j, v := range vrfSelections {
		log.WithField("v", v).WithField(" j ", j).Trace("before sort")
	}
	//log.Trace(vrfSelections)
	sort.Sort(vrfSelections)
	for j, v := range vrfSelections {
		if j == d.ann.NbParticipants {
			break
		}
		cp := d.ann.term.GetCampaign(v.addr)
		if cp == nil {
			panic("cp is nil")
		}
		d.ann.term.AddCandidate(cp)
		log.WithField("v", v).WithField(" j ", j).Trace("you are lucky one")
		if bytes.Equal(cp.PublicKey, d.ann.MyAccount.PublicKey.Bytes) {
			log.Debug("congratulation i am a partner of dkg ")
			d.isValidPartner = true
		}
		//add here with sorted
		d.addPartner(cp)
	}
	if !d.isValidPartner {
		log.Debug("unfortunately  i am not a partner of dkg ")

	} else {
		d.dkgOn = true
		d.GenerateDkg()
	}

	//log.Debug(vrfSelections)

	//for _, camp := range camps {
	//	d.partner.PartPubs = append(d.partner.PartPubs, camp.GetDkgPublicKey())
	//	d.partner.addressIndex[camp.Sender()] = len(d.partner.PartPubs) - 1
	//}
}

func (d *Dkg) getDeals() (DealsMap, error) {
	return d.partner.Dkger.Deals()
}

func (d *Dkg) AddPartner(c *types.Campaign) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.addPartner(c)
	return

}

func (d *Dkg) addPartner(c *types.Campaign) {
	d.partner.PartPubs = append(d.partner.PartPubs, c.GetDkgPublicKey())
	if bytes.Equal(c.PublicKey, d.ann.MyAccount.PublicKey.Bytes) {
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

func (d *Dkg) SendGenesisPublicKey() {
	for i := 0; i < len(d.ann.genesisAccounts); i++ {
		msg := &types.MessageConsensusDkgGenesisPublicKey{
			DkgPublicKey: d.myPublicKey,
			PublicKey:    d.ann.MyAccount.PublicKey.Bytes,
		}
		msg.Signature = d.signer.Sign(d.ann.MyAccount.PrivateKey, msg.SignatureTargets()).Bytes
		if i == d.ann.id {
			log.Tracef("escape me %d ", d.ann.id)
			//myself
			d.ann.genesisPkChan <- msg
			continue
		}
		d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgGenesisPublicKey, msg, &d.ann.genesisAccounts[i])
		log.WithField("msg ", msg).WithField("peer ", i).Debug("send pk to ")
	}
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
		if *addr == d.ann.MyAccount.Address {
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
		msg.Signature = d.signer.Sign(d.ann.MyAccount.PrivateKey, msg.SignatureTargets()).Bytes
		msg.PublicKey = d.ann.MyAccount.PublicKey.Bytes
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
			log.Debug("gossip dkg started")
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
			log.Debug()
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
			if responseDeal.Response.Status != vss.StatusApproval {
				log.WithField("deal ", request).WithError(err).Warn("deal process not StatusApproval")
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
			response.Signature = d.signer.Sign(d.ann.MyAccount.PrivateKey, response.SignatureTargets()).Bytes
			response.PublicKey = d.ann.MyAccount.PublicKey.Bytes
			log.WithField("to request ", request).WithField("response ", &response).Debug("will send response")
			//broadcast response to all partner

			for k, v := range d.ann.Candidates() {
				if k == d.ann.MyAccount.Address {
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
			msg.Signature = d.signer.Sign(d.ann.MyAccount.PrivateKey, msg.SignatureTargets()).Bytes
			msg.PublicKey = d.ann.MyAccount.PublicKey.Bytes

			for k, v := range d.ann.Candidates() {
				if k == d.ann.MyAccount.Address {
					continue
				}
				msgCopy := msg
				pk := crypto.PublicKeyFromBytes(d.ann.cryptoType, v.PublicKey)
				go d.ann.Hub.SendToAnynomous(og.MessageTypeConsensusDkgSigSets, &msgCopy, &pk)
			}

			d.addSigsets(d.ann.MyAccount.Address, &types.SigSet{PublicKey: msg.PublicKey, Signature: msg.Signature})
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
			d.addSigsets(addr, &types.SigSet{PublicKey: response.PublicKey, Signature: response.Signature})

			if len(d.blsSigSets) >= d.partner.NbParticipants {
				log.Info("got enough sig sets")
				d.ann.dkgPulicKeyChan <- d.partner.jointPubKey
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

func (d *Dkg) VerifyBlsSig(msg []byte, sig []byte, jointPub []byte) bool {
	pubKey, err := bn256.UnmarshalBinaryPointG2(jointPub)
	if err != nil {
		log.WithError(err).Warn("unmarshal join pubkey error")
		return false
	}
	if !pubKey.Equal(d.partner.jointPubKey) {
		log.WithField("seq pk ", pubKey).WithField("our joint pk ", d.partner.jointPubKey).Warn("different")
		return false
	}
	err = bls.Verify(d.partner.Suite, pubKey, msg, sig)
	if err != nil {
		log.WithField("sig ", hex.EncodeToString(sig)).WithField("s ", d.partner.Suite).WithField("pk", pubKey).WithError(err).Warn("bls verify error")
		return false
	}
	return true
	//todo how to verify when term change
	// d.partner.jointPubKey
}

func (d *Dkg) RecoverAndVerifySignature(sigShares [][]byte, msg []byte, dkgTermId int) (jointSig []byte, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	partner := d.partner
	//if the term id is small ,use former partner to verify
	if dkgTermId < d.TermId {
		partner = d.formerPartner
	}
	partner.SigShares = sigShares
	defer func() {
		partner.SigShares = nil
	}()
	jointSig, err = partner.RecoverSig(msg)
	if err != nil {
		log.Warnf("partner %d cannot recover jointSig with %d sigshares: %s\n",
			partner.Id, len(partner.SigShares), err)
		return nil, err
	}
	log.Debugf("threshold signature from partner %d: %s\n", partner.Id, hexutil.Encode(jointSig))
	// verify if JointSig meets the JointPubkey
	err = partner.VerifyByDksPublic(msg, jointSig)
	if err != nil {
		log.WithError(err).Warnf("joinsig verify failed ")
		return nil, err
	}

	// verify if JointSig meets the JointPubkey
	err = partner.VerifyByPubPoly(msg, jointSig)
	if err != nil {
		log.WithError(err).Warnf("joinsig verify failed ")
		return nil, err
	}
	return jointSig, nil

}
