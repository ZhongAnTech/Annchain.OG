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
package archive

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/hexutil"
	dkg2 "github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/prometheus/common/log"
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

// DkgPartner is the parter in a DKG group built to discuss a pub/privkey
// It will receive DKG messages and update the status.
// It is the handler for maintaining the DkgContext.
// Campaign or term change is not part of DKGPartner. Do their job in their own module.
type DkgPartner struct {
	//TermId            uint32
	//dkgOn             bool
	myPublircKey      []byte
	context           *dkg2.DkgContext
	formerContext     *dkg2.DkgContext //used for case : context reset , but bft is still generating sequencer
	gossipStartCh     chan struct{}
	gossipStopCh      chan struct{}
	gossipReqCh       chan *p2p_message.MessageConsensusDkgDeal
	gossipRespCh      chan *p2p_message.MessageConsensusDkgDealResponse
	gossipSigSetspCh  chan *p2p_message.MessageConsensusDkgSigSets
	dealCache         map[common.Address]*p2p_message.MessageConsensusDkgDeal
	dealResponseCache map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse
	dealSigSetsCache  map[common.Address]*p2p_message.MessageConsensusDkgSigSets
	respWaitingCache  map[uint32][]*p2p_message.MessageConsensusDkgDealResponse
	blsSigSets        map[common.Address]*tx_types.SigSet
	ready             bool
	isValidPartner    bool

	mu        sync.RWMutex
	myAccount *account.Account
	term      *term.Term
	Hub       dkg2.P2PSender

	OnDkgPublicKeyJointChan chan kyber.Point // joint pubkey is got
	OnGenesisPkChan         chan *p2p_message.MessageConsensusDkgGenesisPublicKey
	ConfigFilePath          string
}

func NewDkgPartner(dkgOn bool, numParts, threshold int, dkgPublicKeyJointChan chan kyber.Point,
	genesisPkChan chan *p2p_message.MessageConsensusDkgGenesisPublicKey, t *term.Term) *DkgPartner {
	// init dkg context to store keys
	c := dkg2.NewDkgContext(bn256.NewSuiteG2())
	c.NbParticipants = numParts
	c.Threshold = threshold
	c.PartPubs = []kyber.Point{}

	d := &DkgPartner{}
	d.context = c

	// message channels
	d.gossipStartCh = make(chan struct{})
	d.gossipReqCh = make(chan *p2p_message.MessageConsensusDkgDeal, 100)
	d.gossipRespCh = make(chan *p2p_message.MessageConsensusDkgDealResponse, 100)
	d.gossipSigSetspCh = make(chan *p2p_message.MessageConsensusDkgSigSets, 100)
	d.gossipStopCh = make(chan struct{})

	d.dkgOn = dkgOn
	if d.dkgOn {
		d.GenerateDkg() //todo fix later
		d.myPublicKey = d.context.CandidatePublicKey[0]
		d.context.MyPartSec = d.context.CandidatePartSec[0]
	}


	d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	d.dealResponseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
	d.blsSigSets = make(map[common.Address]*tx_types.SigSet)
	d.ready = false
	d.isValidPartner = false

	d.OnDkgPublicKeyJointChan = dkgPublicKeyJointChan
	d.OnGenesisPkChan = genesisPkChan
	d.term = t

	return d
}

//func (d *DkgPartner) SetId(id int) {
//	d.context.Id = uint32(id)
//}
//
//func (d *DkgPartner) SetAccount(myAccount *account.Account) {
//	d.myAccount = myAccount
//}
//
//func (d *DkgPartner) GetParticipantNumber() int {
//	return d.context.NbParticipants
//}
//
//func (d *DkgPartner) Reset(myCampaign *tx_types.Campaign) {
//	if myCampaign == nil {
//		log.Warn("nil campagin, I am not a dkg context")
//	}
//
//	d.mu.RLock()
//	defer d.mu.RUnlock()
//	partSecs := d.context.CandidatePartSec
//	pubKeys := d.context.CandidatePublicKey
//	//TODO: keep multiple history generations
//	d.formerContext = d.context
//	d.formerContext.CandidatePartSec = nil
//	d.formerContext.CandidatePublicKey = nil
//	p := NewDkgContext(bn256.NewSuiteG2())
//	p.NbParticipants = d.context.NbParticipants
//	p.Threshold = d.context.Threshold
//	p.PartPubs = []kyber.Point{}
//	if myCampaign != nil {
//		index := -1
//		if len(myCampaign.DkgPublicKey) != 0 {
//			for i, pubKeys := range pubKeys {
//				if bytes.Equal(pubKeys, myCampaign.DkgPublicKey) {
//					index = i
//				}
//			}
//		}
//		if index < 0 {
//			log.WithField("cp", myCampaign).WithField("pks", d.context.CandidatePublicKey).Warn("pk not found")
//			index = 0
//			panic(fmt.Sprintf("fix this, pk not found  %s ", myCampaign))
//		}
//		if index >= len(partSecs) {
//			panic(fmt.Sprint(index, partSecs, hexutil.Encode(myCampaign.DkgPublicKey)))
//		}
//		log.WithField("cp ", myCampaign).WithField("index ", index).WithField("sk ", partSecs[index]).Debug("reset with sk")
//		p.MyPartSec = partSecs[index]
//		p.CandidatePartSec = append(p.CandidatePartSec, partSecs[index:]...)
//		d.myPublicKey = pubKeys[index]
//		p.CandidatePublicKey = append(p.CandidatePublicKey, pubKeys[index:]...)
//	} else {
//		//
//	}
//	d.dkgOn = false
//	d.context = p
//	d.TermId++
//
//	//d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
//	//d.dealResponseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
//	//d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
//	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
//	d.blsSigSets = make(map[common.Address]*tx_types.SigSet)
//	d.ready = false
//	log.WithField("len candidate pk", len(d.context.CandidatePublicKey)).WithField("termId", d.TermId).Debug("dkg will reset")
//}

func (d *DkgPartner) Start() {
	//TODO
	dkg2.log.Info("dkg start")
	goroutine.New(d.gossiploop)
}

func (d *DkgPartner) Stop() {
	d.gossipStopCh <- struct{}{}
	dkg2.log.Info("dkg stoped")
}

//func (d *DkgPartner) Log() *logrus.Entry {
//	return d.log()
//}

//func (d *DkgPartner) On() {
//	d.dkgOn = true
//}

//func (d *DkgPartner) IsValidPartner() bool {
//	return d.isValidPartner
//}

//func (d *DkgPartner) generateDkg() []byte {
//	sec, pub := GenPartnerPair(d.context)
//	d.context.CandidatePartSec = append(d.context.CandidatePartSec, sec)
//	//d.context.PartPubs = []kyber.Point{pub}??
//	pk, _ := pub.MarshalBinary()
//	d.context.CandidatePublicKey = append(d.context.CandidatePublicKey, pk)
//	log.WithField("pk", hexutil.Encode(pk[:5])).
//		WithField("len pk", len(d.context.CandidatePublicKey)).
//		WithField("sk", sec).
//		Debug("gen dkg")
//	return pk
//}

//func (d *DkgPartner) log() *logrus.Entry {
//	return log.WithField("me", d.context.Id).WithField("termId", d.TermId)
//}

//func (d *DkgPartner) GenerateDkg() []byte {
//	d.mu.RLock()
//	defer d.mu.RUnlock()
//	return d.generateDkg()
//}

//PublicKey current pk
//func (d *DkgPartner) PublicKey() []byte {
//	return d.myPublicKey
//}

func (d *DkgPartner) StartGossip() {
	d.gossipStartCh <- struct{}{}
}

//calculate seed
//func CalculateRandomSeed(jointSig []byte) []byte {
//	//TODO
//	h := sha256.New()
//	h.Write(jointSig)
//	seed := h.Sum(nil)
//	return seed
//}

func (d *DkgPartner) getDeals() (DealsMap, error) {
	return d.context.Dkger.Deals()
}

//func (d *DkgPartner) AddPartner(c *tx_types.Campaign) {
//	d.mu.Lock()
//	defer d.mu.Unlock()
//	publicKey := crypto.Signer.PublicKeyFromBytes(c.PublicKey)
//	d.addPartner(c,publicKey)
//	return
//
//}

func (d *DkgPartner) addPartner(c *tx_types.Campaign) {
	d.context.PartPubs = append(d.context.PartPubs, c.GetDkgPublicKey())
	if bytes.Equal(c.PublicKey, d.myAccount.PublicKey.Bytes) {
		d.context.Id = uint32(len(d.context.PartPubs) - 1)
		dkg2.log.WithField("id ", d.context.Id).Trace("my id")
	}
	dkg2.log.WithField("cp ", c).Trace("added context")
	d.context.addressIndex[c.Sender()] = len(d.context.PartPubs) - 1
}

func (d *DkgPartner) GetBlsSigsets() []*tx_types.SigSet {
	var sigset []*tx_types.SigSet
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, sig := range d.blsSigSets {
		sigset = append(sigset, sig)
	}
	return sigset
}

func (d *DkgPartner) GenerateDKGer() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.context.GenerateDKGer()
}

func (d *DkgPartner) ProcesssDeal(dd *dkg.Deal) (resp *dkg.Response, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.context.Dkger.ProcessDeal(dd)
}

func (d *DkgPartner) CheckAddress(addr common.Address) bool {
	if d.term.GetCandidate(addr) == nil {
		return false
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.context.addressIndex[addr]
	return ok
}

func (d *DkgPartner) SendGenesisPublicKey(genesisAccounts []crypto.PublicKey) {
	log := d.log()
	for i := 0; i < len(genesisAccounts); i++ {
		msg := &p2p_message.MessageConsensusDkgGenesisPublicKey{
			DkgPublicKey: d.myPublicKey,
			PublicKey:    d.myAccount.PublicKey.Bytes,
		}
		msg.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, msg.SignatureTargets()).Bytes
		if uint32(i) == d.context.Id {
			log.Tracef("escape me %d ", d.context.Id)
			//myself
			d.OnGenesisPkChan <- msg
			continue
		}
		dkg2.AnonymousSendMessage(message.MessageTypeConsensusDkgGenesisPublicKey, msg, &genesisAccounts[i])
		log.WithField("msg", msg).WithField("peer", i).Debug("send genesis pk to")
	}
}

func (d *DkgPartner) getUnhandled() ([]*p2p_message.MessageConsensusDkgDeal, []*p2p_message.MessageConsensusDkgDealResponse) {
	var unhandledDeal []*p2p_message.MessageConsensusDkgDeal
	var unhandledResponse []*p2p_message.MessageConsensusDkgDealResponse
	d.mu.RLock()
	defer d.mu.RUnlock()
	for i := range d.dealCache {
		unhandledDeal = append(unhandledDeal, d.dealCache[i])
	}
	for i := range d.dealResponseCache {
		unhandledResponse = append(unhandledResponse, d.dealResponseCache[i]...)
	}
	if len(d.dealCache) == 0 {
		d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	}
	if len(d.dealResponseCache) == 0 {
		d.dealResponseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	}
	return unhandledDeal, unhandledResponse
}

func (d *DkgPartner) sendUnhandledTochan(unhandledDeal []*p2p_message.MessageConsensusDkgDeal, unhandledResponse []*p2p_message.MessageConsensusDkgDealResponse) {
	if len(unhandledDeal) != 0 || len(unhandledResponse) != 0 {
		dkg2.log.WithField("unhandledDeal deals ", len(unhandledDeal)).WithField(
			"unhandledResponse", unhandledResponse).Debug("will process")

		for _, v := range unhandledDeal {
			d.gossipReqCh <- v
		}
		for _, v := range unhandledResponse {
			d.gossipRespCh <- v
		}
		dkg2.log.WithField("unhandledDeal deals", len(unhandledDeal)).WithField(
			"unhandledResponse", len(unhandledResponse)).Debug("processed")
	}
}

func (d *DkgPartner) unhandledSigSets() []*p2p_message.MessageConsensusDkgSigSets {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var sigs []*p2p_message.MessageConsensusDkgSigSets
	for _, v := range d.dealSigSetsCache {
		sigs = append(sigs, v)
	}
	d.dealSigSetsCache = make(map[common.Address]*p2p_message.MessageConsensusDkgSigSets)
	return sigs
}

func (d *DkgPartner) ProcessDeal(deal *dkg.Deal) (*dkg.Response, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	responseDeal, err := d.context.Dkger.ProcessDeal(deal)
	if err == nil {
		d.context.dealsIndex[deal.Index] = true
	}
	return responseDeal, err

}

//ProcessWaitingResponse
func (d *DkgPartner) ProcessWaitingResponse(deal *dkg.Deal) {
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

//func (d *DkgPartner) sendDealsToCorrespondingPartner(deals DealsMap, termId uint32) {
//	//for generating deals, we use n partPubs, including our partPub. it generates  n-1 deals, excluding our own deal
//	//skip myself
//	log := d.log()
//	for i, deal := range deals {
//		data, _ := deal.MarshalMsg(nil)
//		msg := &p2p_message.MessageConsensusDkgDeal{
//			Data: data,
//			Id:   message.MsgCounter.Get(),
//		}
//		addr := d.GetPartnerAddressByIndex(i)
//		if addr == nil {
//			panic("address not found")
//		}
//		if *addr == d.myAccount.Address {
//			//this is for me , skip myself
//			log.WithField("i ", i).WithField("to ", addr.TerminalString()).WithField("deal",
//				deal.TerminateString()).Error("send dkg deal to myself")
//			panic("send dkg deal to myself")
//		}
//		//not a candidate
//		cp := d.term.GetCandidate(*addr)
//		if cp == nil {
//			panic("campaign not found")
//		}
//		msg.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, msg.SignatureTargets()).Bytes
//		msg.PublicKey = d.myAccount.PublicKey.Bytes
//		pk := crypto.Signer.PublicKeyFromBytes(cp.PublicKey)
//		log.WithField("to ", addr.TerminalString()).WithField("deal",
//			deal.TerminateString()).WithField("msg", msg).Debug("send dkg deal to")
//		dkg2.AnonymousSendMessage(message.MessageTypeConsensusDkgDeal, msg, &pk)
//	}
//	return
//}

func (d *DkgPartner) GetId() uint32 {
	if d.context == nil {
		return 0
	}
	return d.context.Id
}

func (d *DkgPartner) gossiploop() {
	for {
		select {
		case <-d.gossipStartCh:
		//	log := d.log()
		//	log.Debug("gossip dkg started")
		//	if !d.dkgOn {
		//		//i am not a consensus context
		//		log.Warn("why send to me")
		//		continue
		//	}
		//	err := d.GenerateDKGer()
		//	if err != nil {
		//		log.WithField("sk", d.context.MyPartSec).WithField("part pubs", d.context.PartPubs).WithError(err).Error("gen dkger fail")
		//		continue
		//	}
		//	d.mu.RLock()
		//	deals, err := d.getDeals()
		//	termid := d.TermId
		//	d.mu.RUnlock()
		//	if err != nil {
		//		log.WithError(err).Error("generate dkg deal error")
		//		continue
		//	}
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
				//not a consensus context
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
				log.WithField("deal ", request).Warn("not found  dkg context address  or campaign  for deal")
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

			//response := p2p_message.MessageConsensusDkgDealResponse{
			//	Data: respData,
			//	//Id:   request.Id,
			//	Id: message.MsgCounter.Get(),
			//}
			//response.Signature = crypto.Signer.Sign(d.myAccount.PrivateKey, response.SignatureTargets()).Bytes
			//response.PublicKey = d.myAccount.PublicKey.Bytes
			//log.WithField("to request ", request).WithField("response ", &response).Debug("will send response")
			//broadcast response to all context

			//for k, v := range d.term.Candidates() {
			//	if k == d.myAccount.Address {
			//		continue
			//	}
			//	msgCopy := response
			//	pk := crypto.Signer.PublicKeyFromBytes(v.PublicKey)
			//	goroutine.New(func() {
			//		dkg2.AnonymousSendMessage(message.MessageTypeConsensusDkgDealResponse, &msgCopy, &pk)
			//	})
			//}
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
				//not a consensus context
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
					"not found  dkg  context or campaign msg for address  of this  deal")
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
			d.context.responseNumber++
			d.mu.RUnlock()
			log.WithField("response number", d.context.responseNumber).Trace("dkg")
			//will got  (n-1)*(n-1) response
			if d.context.responseNumber < (d.context.NbParticipants-1)*(d.context.NbParticipants-1) {
				continue
			}
			log.Debug("got response done")
			d.mu.RLock()
			jointPub, err := d.context.RecoverPub()
			d.mu.RUnlock()
			if err != nil {
				log.WithError(err).Warn("get recover pub key fail")
				continue
			}
			// send public key to changeTerm loop.
			// TODO
			// this channel may be changed later.
			log.WithField("bls key", jointPub).Trace("joint pubkey")
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
					dkg2.AnonymousSendMessage(message.MessageTypeConsensusDkgSigSets, &msgCopy, &pk)
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
				//not a consensus context
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
					"not found  dkg  context or campaign msg for address  of this  deal")
				continue
			}
			if !pkBls.Equal(d.context.JointPubKey) {
				log.WithField("got pkbls ", pkBls).WithField("joint pk ", d.context.JointPubKey).Warn("pk bls mismatch")
				continue
			}
			d.addSigsets(addr, &tx_types.SigSet{PublicKey: response.PublicKey, Signature: response.Signature})

			if len(d.blsSigSets) >= d.context.NbParticipants {
				log.Info("got enough sig sets")
				d.context.KeyShare, err = d.context.Dkger.DistKeyShare()
				if err != nil {
					log.WithError(err).Error("key share err")
				}
				d.OnDkgPublicKeyJointChan <- d.context.JointPubKey
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

func (d *DkgPartner) clearCache() {
	d.dealCache = make(map[common.Address]*p2p_message.MessageConsensusDkgDeal)
	d.dealResponseCache = make(map[common.Address][]*p2p_message.MessageConsensusDkgDealResponse)
	d.respWaitingCache = make(map[uint32][]*p2p_message.MessageConsensusDkgDealResponse)
}

//CacheResponseIfDealNotReceived
func (d *DkgPartner) CacheResponseIfDealNotReceived(resp *dkg.Response, response *p2p_message.MessageConsensusDkgDealResponse) (ok bool) {
	log := d.log()
	d.mu.RLock()
	defer d.mu.RUnlock()
	if v := d.context.dealsIndex[resp.Index]; !v && resp.Index != d.context.Id {
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
func (d *DkgPartner) CacheResponseIfNotReady(addr common.Address, response *p2p_message.MessageConsensusDkgDealResponse) (ready bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ready {
		dkgResps, _ := d.dealResponseCache[addr]
		dkgResps = append(dkgResps, response)
		d.dealResponseCache[addr] = dkgResps
	}
	return d.ready
}

//CacheDealsIfNotReady check whether dkg is ready, not ready  cache the dkg deal
func (d *DkgPartner) CacheDealsIfNotReady(addr common.Address, request *p2p_message.MessageConsensusDkgDeal) (ready bool) {
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
func (d *DkgPartner) CacheSigSetsIfNotReady(addr common.Address, response *p2p_message.MessageConsensusDkgSigSets) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if !d.ready || d.context.JointPubKey == nil {
		if _, ok := d.dealSigSetsCache[addr]; !ok {
			d.dealSigSetsCache[addr] = response
		}
		return false

	}
	return true
}

func (d *DkgPartner) addSigsets(addr common.Address, sig *tx_types.SigSet) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.blsSigSets[addr] = sig
	return

}

func (d *DkgPartner) ProcessResponse(resp *dkg.Response) (just *dkg.Justification, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.processResponse(resp)
}

func (d *DkgPartner) processResponse(resp *dkg.Response) (just *dkg.Justification, err error) {
	return d.context.Dkger.ProcessResponse(resp)
}

func (d *DkgPartner) GetPartnerAddressByIndex(i int) *common.Address {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for k, v := range d.context.addressIndex {
		if v == i {
			return &k
		}
	}
	return nil
}

func (d *DkgPartner) GetAddresIndex() map[common.Address]int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.context.addressIndex
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

func (d *DkgPartner) VerifyBlsSig(msg []byte, sig []byte, jointPub []byte, termId uint32) bool {
	log := d.log()
	pubKey, err := bn256.UnmarshalBinaryPointG2(jointPub)
	if err != nil {
		log.WithError(err).Warn("unmarshal join pubkey error")
		return false
	}
	if termId < d.TermId {
		if !pubKey.Equal(d.formerContext.JointPubKey) {
			log.WithField("termId ", termId).WithField("seq pk ", pubKey).WithField("our joint pk ", d.formerContext.JointPubKey).Warn("different")
			return false
		}
		err = bls.Verify(d.formerContext.Suite, pubKey, msg, sig)
		if err != nil {
			log.WithField("sig ", hex.EncodeToString(sig)).WithField("s ", d.context.Suite).WithField("pk", pubKey).WithError(err).Warn("bls verify error")
			return false
		}
		return true
	}
	if !pubKey.Equal(d.context.JointPubKey) {
		log.WithField("seq pk ", pubKey).WithField("our joint pk ", d.context.JointPubKey).Warn("different")
		return false
	}
	err = bls.Verify(d.context.Suite, pubKey, msg, sig)
	if err != nil {
		log.WithField("sig ", hex.EncodeToString(sig)).WithField("s ", d.context.Suite).WithField("pk", pubKey).WithError(err).Warn("bls verify error")
		return false
	}
	return true
	//Todo how to verify when term change
	// d.context.JointPubKey
}

func (d *DkgPartner) Sign(msg []byte, termId uint32) (partSig []byte, err error) {
	partner := d.context
	if termId < d.TermId {
		partner = d.formerContext
		dkg2.log.Trace("use former context to sign")
	}
	return partner.PartSig(msg)
}

func (d *DkgPartner) GetJoinPublicKey(termId uint32) kyber.Point {
	partner := d.context
	if termId < d.TermId {
		partner = d.formerContext
		dkg2.log.Trace("use former context to sign")
	}
	return partner.JointPubKey
}

func (d *DkgPartner) RecoverAndVerifySignature(sigShares [][]byte, msg []byte, dkgTermId uint32) (jointSig []byte, err error) {
	log := d.log()
	d.mu.RLock()
	defer d.mu.RUnlock()
	partner := d.context
	//if the term id is small ,use former context to verify
	if dkgTermId < d.TermId {
		partner = d.formerContext
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
		log.WithField("sigStr ", sigStr).WithField("msg ", hexutil.Encode(msg)).Warnf("context %d cannot recover jointSig with %d sigshares: %s\n",
			partner.Id, len(partner.SigShares), err)
		return nil, err
	}
	log.Tracef("threshold signature from context %d: %s\n", partner.Id, hexutil.Encode(jointSig))
	// verify if JointSig meets the JointPubkey
	err = partner.VerifyByDksPublic(msg, jointSig)
	if err != nil {
		log.WithError(err).Warnf("joinsig verify failed")
		return nil, err
	}

	// verify if JointSig meets the JointPubkey
	err = partner.VerifyByPubPoly(msg, jointSig)
	if err != nil {
		log.WithError(err).Warnf("joinsig verify failed")
		return nil, err
	}
	return jointSig, nil

}

func (d *DkgPartner) SetJointPk(pk kyber.Point) {
	d.context.JointPubKey = pk
}

type DKGInfo struct {
	TermId             uint32                 `json:"term_id"`
	Id                 uint32                 `json:"id"`
	PartPubs           []kyber.Point          `json:"part_pubs"`
	MyPartSec          kyber.Scalar           `json:"-"`
	CandidatePartSec   []kyber.Scalar         `json:"-"`
	CandidatePublicKey []hexutil.Bytes        `json:"candidate_public_key"`
	AddressIndex       map[common.Address]int `json:"address_index"`
}

func (dkg *DkgPartner) GetInfo() *DKGInfo {
	dkgInfo := DKGInfo{
		TermId:           dkg.TermId,
		Id:               dkg.context.Id,
		PartPubs:         dkg.context.PartPubs,
		MyPartSec:        dkg.context.MyPartSec,
		CandidatePartSec: dkg.context.CandidatePartSec,
		AddressIndex:     dkg.context.addressIndex,
	}
	for _, v := range dkg.context.CandidatePublicKey {
		dkgInfo.CandidatePublicKey = append(dkgInfo.CandidatePublicKey, v)
	}
	return &dkgInfo
}

func (d *DkgPartner) HandleDkgDeal(request *p2p_message.MessageConsensusDkgDeal) {
	d.gossipReqCh <- request
}

func (d *DkgPartner) HandleDkgDealRespone(response *p2p_message.MessageConsensusDkgDealResponse) {
	d.gossipRespCh <- response
}

func (d *DkgPartner) HandleSigSet(requset *p2p_message.MessageConsensusDkgSigSets) {
	d.gossipSigSetspCh <- requset
}
