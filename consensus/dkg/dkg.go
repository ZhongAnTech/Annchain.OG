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
package dkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/annsensus/announcer"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"sync"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/annchain/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/kyber/v3/share/vss/pedersen"
	"github.com/annchain/kyber/v3/sign/bls"
)

type Dkg struct {
	TermId            uint32
	dkgOn             bool
	myPublicKey       []byte
	partner           *DKGPartner
	formerPartner     *DKGPartner //used for case : partner reset , but bft is still generating sequencer
	gossipStartCh     chan struct{}
	gossipStopCh      chan struct{}
	gossipReqCh       chan *p2p_message.MessageConsensusDkgDeal
	gossipRespCh      chan *p2p_message.MessageConsensusDkgDealResponse
	gossipSigSetspCh  chan *p2p_message.MessageConsensusDkgSigSets
	dealCache         map[common.Address]*p2p_message.MessageConsensusDkgDeal
	dealResPonseCache map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse
	dealSigSetsCache  map[common.Address]*p2p_message.MessageConsensusDkgSigSets
	respWaitingCache  map[uint32][]*p2p_message.MessageConsensusDkgDealResponse
	blsSigSets        map[common.Address]*tx_types.SigSet
	ready             bool
	isValidPartner    bool

	mu        sync.RWMutex
	myAccount *account.Account
	term      *term.Term
	Hub       announcer.MessageSender

	OndkgPulicKeyChan chan kyber.Point
	OngenesisPkChan   chan *p2p_message.MessageConsensusDkgGenesisPublicKey
	ConfigFilePath    string
}

func NewDkg(dkgOn bool, numParts, threshold int,
	dkgPulicKeyChan chan kyber.Point, genesisPkChan chan *p2p_message.MessageConsensusDkgGenesisPublicKey, t *term.Term) *Dkg {
	p := NewDKGPartner(bn256.NewSuiteG2())
	p.NbParticipants = numParts
	p.Threshold = threshold
	p.PartPubs = []kyber.Point{}

	d := &Dkg{}
	d.partner = p
	d.gossipStartCh = make(chan struct{})
	d.gossipReqCh = make(chan *p2p_message.MessageConsensusDkgDeal, 100)
	d.gossipRespCh = make(chan *p2p_message.MessageConsensusDkgDealResponse, 100)
	d.gossipSigSetspCh = make(chan *p2p_message.MessageConsensusDkgSigSets, 100)
	d.gossipStopCh = make(chan struct{})
	d.dkgOn = dkgOn
	if d.dkgOn {
		d.GenerateDkg() //todo fix later
		d.myPublicKey = d.partner.CandidatePublicKey[0]
		d.partner.MyPartSec = d.partner.CandidatePartSec[0]
	}
	d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	d.dealResPonseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[common.Address]*tx_types.SigSet)
	d.ready = false
	d.isValidPartner = false
	d.OndkgPulicKeyChan = dkgPulicKeyChan
	d.OngenesisPkChan = genesisPkChan
	d.term = t

	return d
}

func (d *Dkg) SetId(id int) {
	d.partner.Id = uint32(id)
}

func (d *Dkg) SetAccount(myAccount *account.Account) {
	d.myAccount = myAccount
}

func (d *Dkg) GetParticipantNumber() int {
	return d.partner.NbParticipants
}

func (d *Dkg) Reset(myCampaign *tx_types.Campaign) {
	if myCampaign == nil {
		log.Warn("nil campagin,  i am not a dkg partner")
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	partSecs := d.partner.CandidatePartSec
	pubKeys := d.partner.CandidatePublicKey
	d.formerPartner = d.partner
	d.formerPartner.CandidatePartSec = nil
	d.formerPartner.CandidatePublicKey = nil
	p := NewDKGPartner(bn256.NewSuiteG2())
	p.NbParticipants = d.partner.NbParticipants
	p.Threshold = d.partner.Threshold
	p.PartPubs = []kyber.Point{}
	if myCampaign != nil {
		index := -1
		if len(myCampaign.DkgPublicKey) != 0 {
			for i, pubKeys := range pubKeys {
				if bytes.Equal(pubKeys, myCampaign.DkgPublicKey) {
					index = i
				}
			}
		}
		if index < 0 {
			log.WithField("cp ", myCampaign).WithField(" pks ", d.partner.CandidatePublicKey).Warn("pk not found")
			index = 0
			panic(fmt.Sprintf("fix this, pk not found  %s ", myCampaign))
		}
		if index >= len(partSecs) {
			panic(fmt.Sprint(index, partSecs, hexutil.Encode(myCampaign.DkgPublicKey)))
		}
		log.WithField("cp ", myCampaign).WithField("index ", index).WithField("sk ", partSecs[index]).Debug("reset with sk")
		p.MyPartSec = partSecs[index]
		p.CandidatePartSec = append(p.CandidatePartSec, partSecs[index:]...)
		d.myPublicKey = pubKeys[index]
		p.CandidatePublicKey = append(p.CandidatePublicKey, pubKeys[index:]...)
	} else {
		//
	}
	d.dkgOn = false
	d.partner = p
	d.TermId++

	//d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	//d.dealResPonseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	//d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[common.Address]*tx_types.SigSet)
	d.ready = false
	log.WithField("len candidate pk ", len(d.partner.CandidatePublicKey)).WithField("termId ", d.TermId).Debug("dkg will reset")
}

func (d *Dkg) Start() {
	//TODO
	log.Info("dkg start")
	goroutine.New(d.gossiploop)
}

func (d *Dkg) Stop() {
	d.gossipStopCh <- struct{}{}
	log.Info("dkg stoped")
}

func (d *Dkg) Log() *logrus.Entry {
	return d.log()
}

func (d *Dkg) On() {
	d.dkgOn = true
}

func (d *Dkg) IsValidPartner() bool {
	return d.isValidPartner
}

func (d *Dkg) generateDkg() []byte {
	sec, pub := genPartnerPair(d.partner)
	d.partner.CandidatePartSec = append(d.partner.CandidatePartSec, sec)
	//d.partner.PartPubs = []kyber.Point{pub}??
	pk, _ := pub.MarshalBinary()
	d.partner.CandidatePublicKey = append(d.partner.CandidatePublicKey, pk)
	log.WithField("pk ", hexutil.Encode(pk[:5])).WithField("len pk ", len(d.partner.CandidatePublicKey)).WithField("sk ", sec).Debug("gen dkg ")
	return pk
}

func (d *Dkg) log() *logrus.Entry {
	return log.WithField("me ", d.partner.Id).WithField("termId ", d.TermId)
}

func (d *Dkg) GenerateDkg() []byte {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.generateDkg()
}

//PublicKey current pk
func (d *Dkg) PublicKey() []byte {
	return d.myPublicKey
}

func (d *Dkg) StartGossip() {
	d.gossipStartCh <- struct{}{}
}

type VrfSelection struct {
	addr   common.Address
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

func (d *Dkg) SelectCandidates(seq *tx_types.Sequencer) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	defer func() {
		//set nil after select
		d.term.ClearCampaigns()
	}()
	log := d.log()
	campaigns := d.term.Campaigns()
	if len(campaigns) == d.partner.NbParticipants {
		log.Debug("campaign number is equal to participant number ,all will be senator")
		var txs types.Txis
		for _, cp := range campaigns {
			if bytes.Equal(cp.PublicKey, d.myAccount.PublicKey.Bytes) {
				d.isValidPartner = true
				d.dkgOn = true
			}
			txs = append(txs, cp)
		}
		sort.Sort(txs)
		log.WithField("txs ", txs).Debug("lucky cps")
		for _, tx := range txs {
			cp := tx.(*tx_types.Campaign)
			publicKey := crypto.Signer.PublicKeyFromBytes(cp.PublicKey)
			d.term.AddCandidate(cp, publicKey)
			if d.isValidPartner {
				d.addPartner(cp)
			}
		}

		if d.isValidPartner {
			//d.generateDkg()
			log.Debug("you are lucky one")
		}
		return
	}
	if len(campaigns) < d.partner.NbParticipants {
		panic("never come here , programmer error")
	}
	randomSeed := CalculateRandomSeed(seq.BlsJointSig)
	log.WithField("rand seed ", hexutil.Encode(randomSeed)).Debug("generated")
	var vrfSelections VrfSelections
	var j int
	for addr, cp := range campaigns {
		vrfSelect := VrfSelection{
			addr:   addr,
			Vrf:    cp.Vrf.Vrf,
			XORVRF: XOR(cp.Vrf.Vrf, randomSeed),
			Id:     j,
		}
		j++
		vrfSelections = append(vrfSelections, vrfSelect)
	}
	log.Debugf("we have %d capmpaigns, select %d of them ", len(campaigns), d.partner.NbParticipants)
	for j, v := range vrfSelections {
		log.WithField("v", v).WithField(" j ", j).Trace("before sort")
	}
	//log.Trace(vrfSelections)
	sort.Sort(vrfSelections)
	log.WithField("txs ", vrfSelections).Debug("lucky cps")
	for j, v := range vrfSelections {
		if j == d.partner.NbParticipants {
			break
		}
		cp := d.term.GetCampaign(v.addr)
		if cp == nil {
			panic("cp is nil")
		}
		publicKey := crypto.Signer.PublicKeyFromBytes(cp.PublicKey)
		d.term.AddCandidate(cp, publicKey)
		log.WithField("v", v).WithField(" j ", j).Trace("you are lucky one")
		if bytes.Equal(cp.PublicKey, d.myAccount.PublicKey.Bytes) {
			log.Debug("congratulation i am a partner of dkg ")
			d.isValidPartner = true
			d.partner.Id = uint32(j)
		}
		//add here with sorted
		d.addPartner(cp)
	}
	if !d.isValidPartner {
		log.Debug("unfortunately  i am not a partner of dkg ")
	} else {
		d.dkgOn = true
		//d.generateDkg()
	}

	//log.Debug(vrfSelections)

	//for _, camp := range camps {
	//	d.partner.PartPubs = append(d.partner.PartPubs, camp.GetDkgPublicKey())
	//	d.partner.addressIndex[camp.Sender()] = len(d.partner.PartPubs) - 1
	//}
}

//calculate seed
func CalculateRandomSeed(jointSig []byte) []byte {
	//TODO
	h := sha256.New()
	h.Write(jointSig)
	seed := h.Sum(nil)
	return seed
}

func (d *Dkg) getDeals() (DealsMap, error) {
	return d.partner.Dkger.Deals()
}

//func (d *Dkg) AddPartner(c *tx_types.Campaign) {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//	publicKey := crypto.Signer.PublicKeyFromBytes(c.PublicKey)
//	d.addPartner(c,publicKey)
//	return
//
//}

func (d *Dkg) addPartner(c *tx_types.Campaign) {
	d.partner.PartPubs = append(d.partner.PartPubs, c.GetDkgPublicKey())
	if bytes.Equal(c.PublicKey, d.myAccount.PublicKey.Bytes) {
		d.partner.Id = uint32(len(d.partner.PartPubs) - 1)
		log.WithField("id ", d.partner.Id).Trace("my id")
	}
	log.WithField("cp ", c).Trace("added partner")
	d.partner.addressIndex[c.Sender()] = len(d.partner.PartPubs) - 1
}

func (d *Dkg) GetBlsSigsets() []*tx_types.SigSet {
	var sigset []*tx_types.SigSet
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

func (d *Dkg) CheckAddress(addr common.Address) bool {
	if d.term.GetCandidate(addr) == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.partner.addressIndex[addr]
	return ok
}

func (d *Dkg) SendGenesisPublicKey(genesisAccounts []crypto.PublicKey) {
	log := d.log()
	for i := 0; i < len(genesisAccounts); i++ {
		msg := &p2p_message.MessageConsensusDkgGenesisPublicKey{
			DkgPublicKey: d.myPublicKey,
			PublicKey:    d.myAccount.PublicKey.Bytes,
		}
		msg.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, msg.SignatureTargets()).Bytes
		if uint32(i) == d.partner.Id {
			log.Tracef("escape me %d ", d.partner.Id)
			//myself
			d.OngenesisPkChan <- msg
			continue
		}
		d.Hub.SendToAnynomous(p2p_message.MessageTypeConsensusDkgGenesisPublicKey, msg, &genesisAccounts[i])
		log.WithField("msg ", msg).WithField("peer ", i).Debug("send genesis pk to ")
	}
}

func (d *Dkg) getUnhandled() ([]*p2p_message.MessageConsensusDkgDeal, []*p2p_message.MessageConsensusDkgDealResponse) {
	var unhandledDeal []*p2p_message.MessageConsensusDkgDeal
	var unhandledResponse []*p2p_message.MessageConsensusDkgDealResponse
	d.mu.RLock()
	defer d.mu.RUnlock()
	for i := range d.dealCache {
		unhandledDeal = append(unhandledDeal, d.dealCache[i])
	}
	for i := range d.dealResPonseCache {
		unhandledResponse = append(unhandledResponse, d.dealResPonseCache[i]...)
	}
	if len(d.dealCache) == 0 {
		d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	}
	if len(d.dealResPonseCache) == 0 {
		d.dealResPonseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	}
	return unhandledDeal, unhandledResponse
}

func (d *Dkg) sendUnhandledTochan(unhandledDeal []*p2p_message.MessageConsensusDkgDeal, unhandledResponse []*p2p_message.MessageConsensusDkgDealResponse) {
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

func (d *Dkg) unhandledSigSets() []*p2p_message.MessageConsensusDkgSigSets {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var sigs []*p2p_message.MessageConsensusDkgSigSets
	for _, v := range d.dealSigSetsCache {
		sigs = append(sigs, v)
	}
	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
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
	log := d.log()
	var resps []*p2p_message.MessageConsensusDkgDealResponse
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

func (d *Dkg) sendDealsToCorrespondingPartner(deals DealsMap, termId uint32) {
	//for generating deals, we use n partPubs, including our partPub. it generates  n-1 deals, excluding our own deal
	//skip myself
	log := d.log()
	for i, deal := range deals {
		data, _ := deal.MarshalMsg(nil)
		msg := &p2p_message.MessageConsensusDkgDeal{
			Data: data,
			Id:   p2p_message.MsgCounter.Get(),
		}
		addr := d.GetPartnerAddressByIndex(i)
		if addr == nil {
			panic("address not found")
		}
		if *addr == d.myAccount.Address {
			//this is for me , skip myself
			log.WithField("i ", i).WithField("to ", addr.TerminalString()).WithField("deal",
				deal.TerminateString()).Error("send dkg deal to myself")
			panic("send dkg deal to myself")
		}
		//not a candidate
		cp := d.term.GetCandidate(*addr)
		if cp == nil {
			panic("campaign not found")
		}
		msg.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, msg.SignatureTargets()).Bytes
		msg.PublicKey = d.myAccount.PublicKey.Bytes
		pk := crypto.Signer.PublicKeyFromBytes(cp.PublicKey)
		log.WithField("to ", addr.TerminalString()).WithField("deal",
			deal.TerminateString()).WithField("msg", msg).Debug("send dkg deal to")
		d.Hub.SendToAnynomous(p2p_message.MessageTypeConsensusDkgDeal, msg, &pk)
	}
	return
}

func (d *Dkg) GetId() uint32 {
	if d.partner == nil {
		return 0
	}
	return d.partner.Id
}

func (d *Dkg) gossiploop() {
	for {
		select {
		case <-d.gossipStartCh:
			log := d.log()
			log.Debug("gossip dkg started")
			if !d.dkgOn {
				//i am not a consensus partner
				log.Warn("why send to me")
				continue
			}
			err := d.GenerateDKGer()
			if err != nil {
				log.WithField("sk ", d.partner.MyPartSec).WithField("part pubs ", d.partner.PartPubs).WithError(err).Error("gen dkger fail")
				continue
			}
			d.mu.RLock()
			deals, err := d.getDeals()
			termid := d.TermId
			d.mu.RUnlock()
			if err != nil {
				log.WithError(err).Error("generate dkg deal error")
				continue
			}
			unhandledDeal, unhandledResponse := d.getUnhandled()
			d.mu.RLock()
			//dkg is ready now, can process dkg msg
			d.ready = true
			d.mu.RUnlock()
			log.Debug("dkg is ready")
			d.sendDealsToCorrespondingPartner(deals, termid)
			//process unhandled(cached) dkg msg
			d.sendUnhandledTochan(unhandledDeal, unhandledResponse)

		case request := <-d.gossipReqCh:
			log := d.log()
			if !d.dkgOn {
				//not a consensus partner
				log.Warn("why send to me")
				continue
			}

			addr := crypto.Signer.AddressFromPubKeyBytes(request.PublicKey)
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
				log.WithField("from address ", addr.TerminalString()).WithField("deal ", request).WithError(err).Warn("deal process error")
				continue
			}
			if responseDeal.Response.Status != vss.StatusApproval {
				log.WithField("deal ", request).WithError(err).Warn("deal process not StatusApproval")
				continue
			}
			respData, err := responseDeal.MarshalMsg(nil)
			if err != nil {
				log.WithField("deal ", request).WithError(err).Warn("deal process marshal error")
				continue
			}

			response := p2p_message.MessageConsensusDkgDealResponse{
				Data: respData,
				//Id:   request.Id,
				Id: p2p_message.MsgCounter.Get(),
			}
			response.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, response.SignatureTargets()).Bytes
			response.PublicKey = d.myAccount.PublicKey.Bytes
			log.WithField("to request ", request).WithField("response ", &response).Debug("will send response")
			//broadcast response to all partner

			for k, v := range d.term.Candidates() {
				if k == d.myAccount.Address {
					continue
				}
				msgCopy := response
				pk := crypto.Signer.PublicKeyFromBytes(v.PublicKey)
				goroutine.New(func() {
					d.Hub.SendToAnynomous(p2p_message.MessageTypeConsensusDkgDealResponse, &msgCopy, &pk)
				})
			}
			//and sent to myself ? already processed inside dkg,skip myself
			d.ProcessWaitingResponse(&deal)

		case response := <-d.gossipRespCh:
			log := d.log()
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
			addr := crypto.Signer.AddressFromPubKeyBytes(response.PublicKey)
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
				log.WithField("addr ", addr.TerminalString()).WithField("for index ", resp.Index).Debug("deal  not received yet")
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
			log.Debug("got response done")
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
			log.WithField("bls key ", jointPub).Trace("joint pubkey ")
			//d.ann.dkgPkCh <- jointPub
			var msg p2p_message.MessageConsensusDkgSigSets
			msg.PkBls, _ = jointPub.MarshalBinary()
			msg.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, msg.SignatureTargets()).Bytes
			msg.PublicKey = d.myAccount.PublicKey.Bytes

			for k, v := range d.term.Candidates() {
				if k == d.myAccount.Address {
					continue
				}
				msgCopy := msg
				pk := crypto.Signer.PublicKeyFromBytes(v.PublicKey)
				goroutine.New(func() {
					d.Hub.SendToAnynomous(p2p_message.MessageTypeConsensusDkgSigSets, &msgCopy, &pk)
				})
			}

			d.addSigsets(d.myAccount.Address, &tx_types.SigSet{PublicKey: msg.PublicKey, Signature: msg.Signature})
			sigCaches := d.unhandledSigSets()
			for _, sigSets := range sigCaches {
				d.gossipSigSetspCh <- sigSets
			}

		case response := <-d.gossipSigSetspCh:
			log := d.log()
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
			addr := crypto.Signer.AddressFromPubKeyBytes(response.PublicKey)
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
			d.addSigsets(addr, &tx_types.SigSet{PublicKey: response.PublicKey, Signature: response.Signature})

			if len(d.blsSigSets) >= d.partner.NbParticipants {
				log.Info("got enough sig sets")
				d.partner.KeyShare, err = d.partner.Dkger.DistKeyShare()
				if err != nil {
					log.WithError(err).Error("key share err")
				}
				d.OndkgPulicKeyChan <- d.partner.jointPubKey
				d.ready = false
				d.clearCache()

			}

		case <-d.gossipStopCh:
			log := d.log()
			log.Info("got quit signal dkg gossip stopped")
			return
		}
	}
}

func (d *Dkg) clearCache() {
	d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	d.dealResPonseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
}

//CacheResponseIfDealNotReceived
func (d *Dkg) CacheResponseIfDealNotReceived(resp *dkg.Response, response *p2p_message.MessageConsensusDkgDealResponse) (ok bool) {
	log := d.log()
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
func (d *Dkg) CacheResponseIfNotReady(addr common.Address, response *p2p_message.MessageConsensusDkgDealResponse) (ready bool) {
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
func (d *Dkg) CacheDealsIfNotReady(addr common.Address, request *p2p_message.MessageConsensusDkgDeal) (ready bool) {
	log := d.log()
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
func (d *Dkg) CacheSigSetsIfNotReady(addr common.Address, response *p2p_message.MessageConsensusDkgSigSets) bool {
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

func (d *Dkg) addSigsets(addr common.Address, sig *tx_types.SigSet) {
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

func (d *Dkg) GetPartnerAddressByIndex(i int) *common.Address {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for k, v := range d.partner.addressIndex {
		if v == i {
			return &k
		}
	}
	return nil
}

func (d *Dkg) GetAddresIndex() map[common.Address]int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.partner.addressIndex
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

func (d *Dkg) VerifyBlsSig(msg []byte, sig []byte, jointPub []byte, termId int) bool {
	log := d.log()
	pubKey, err := bn256.UnmarshalBinaryPointG2(jointPub)
	if err != nil {
		log.WithError(err).Warn("unmarshal join pubkey error")
		return false
	}
	if termId < d.TermId {
		if !pubKey.Equal(d.formerPartner.jointPubKey) {
			log.WithField("termId ", termId).WithField("seq pk ", pubKey).WithField("our joint pk ", d.formerPartner.jointPubKey).Warn("different")
			return false
		}
		err = bls.Verify(d.formerPartner.Suite, pubKey, msg, sig)
		if err != nil {
			log.WithField("sig ", hex.EncodeToString(sig)).WithField("s ", d.partner.Suite).WithField("pk", pubKey).WithError(err).Warn("bls verify error")
			return false
		}
		return true
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
	//Todo how to verify when term change
	// d.partner.jointPubKey
}

func (d *Dkg) Sign(msg []byte, termId int) (partSig []byte, err error) {
	partner := d.partner
	if termId < d.TermId {
		partner = d.formerPartner
		log.Trace("use former partner to sign")
	}
	return partner.Sig(msg)
}

func (d *Dkg) GetJoinPublicKey(termId int) kyber.Point {
	partner := d.partner
	if termId < d.TermId {
		partner = d.formerPartner
		log.Trace("use former partner to sign")
	}
	return partner.jointPubKey
}

func (d *Dkg) RecoverAndVerifySignature(sigShares [][]byte, msg []byte, dkgTermId uint32) (jointSig []byte, err error) {
	log := d.log()
	d.mu.RLock()
	defer d.mu.RUnlock()
	partner := d.partner
	//if the term id is small ,use former partner to verify
	if dkgTermId < d.TermId {
		partner = d.formerPartner
		log.Trace("use former id")
	}
	partner.SigShares = sigShares
	defer func() {
		partner.SigShares = nil
	}()
	jointSig, err = partner.RecoverSig(msg)
	if err != nil {
		var sigStr string
		for _, ss := range partner.SigShares {
			sigStr = sigStr + " " + hexutil.Encode(ss)
		}
		log.WithField("sigStr ", sigStr).WithField("msg ", hexutil.Encode(msg)).Warnf("partner %d cannot recover jointSig with %d sigshares: %s\n",
			partner.Id, len(partner.SigShares), err)
		return nil, err
	}
	log.Tracef("threshold signature from partner %d: %s\n", partner.Id, hexutil.Encode(jointSig))
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

func (d *Dkg) SetJointPk(pk kyber.Point) {
	d.partner.jointPubKey = pk
}

type DKGInfo struct {
	TermId             int                    `json:"term_id"`
	Id                 uint32                 `json:"id"`
	PartPubs           []kyber.Point          `json:"part_pubs"`
	MyPartSec          kyber.Scalar           `json:"-"`
	CandidatePartSec   []kyber.Scalar         `json:"-"`
	CandidatePublicKey []hexutil.Bytes        `json:"candidate_public_key"`
	AddressIndex       map[common.Address]int `json:"address_index"`
}

func (dkg *Dkg) GetInfo() *DKGInfo {
	dkgInfo := DKGInfo{
		TermId:           dkg.TermId,
		Id:               dkg.partner.Id,
		PartPubs:         dkg.partner.PartPubs,
		MyPartSec:        dkg.partner.MyPartSec,
		CandidatePartSec: dkg.partner.CandidatePartSec,
		AddressIndex:     dkg.partner.addressIndex,
	}
	for _, v := range dkg.partner.CandidatePublicKey {
		dkgInfo.CandidatePublicKey = append(dkgInfo.CandidatePublicKey, v)
	}
	return &dkgInfo
}

func (d *Dkg) HandleDkgDeal(request *p2p_message.MessageConsensusDkgDeal) {
	d.gossipReqCh <- request
}

func (d *Dkg) HandleDkgDealRespone(response *p2p_message.MessageConsensusDkgDealResponse) {
	d.gossipRespCh <- response
}

func (d *Dkg) HandleSigSet(requset *p2p_message.MessageConsensusDkgSigSets) {
	d.gossipSigSetspCh <- requset
}
