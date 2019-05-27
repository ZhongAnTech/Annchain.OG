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
	"errors"
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/hexutil"
	"sync"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

type AnnSensus struct {
	//id           int
	campaignFlag bool
	cryptoType   crypto.CryptoType

	dkg  *Dkg
	term *Term
	bft  *BFT // tendermint protocal

	dkgPulicKeyChan      chan kyber.Point // channel for receiving dkg response.
	newTxHandlers        []chan types.Txi // channels to send txs.
	ConsensusTXConfirmed chan []types.Txi // receiving consensus txs from dag confirm.
	startTermChange      chan bool

	Hub    MessageSender
	Txpool og.ITxPool
	Idag   og.IDag

	MyAccount      *account.SampleAccount
	Threshold      int
	NbParticipants int

	mu sync.RWMutex

	close                         chan struct{}
	genesisAccounts               []crypto.PublicKey
	isGenesisPartner              bool
	genesisBftIsRunning           uint32
	UpdateEvent                   chan bool // syner update event
	newTermChan                   chan bool
	genesisPkChan                 chan *types.MessageConsensusDkgGenesisPublicKey
	NewPeerConnectedEventListener chan string
	ProposalSeqChan               chan types.Hash
	HandleNewTxi                  func(tx types.Txi)
	OnSelfGenTxi                  chan types.Txi

	TxEnable           bool
	NewLatestSequencer chan bool

	ConfigFilePath    string
	termChangeChan    chan *types.TermChange
	currentTermChange *types.TermChange
	disableTermChange bool
	disable           bool
	addedGenesisCampaign  bool
}

func Maj23(n int) int {
	return 2*n/3 + 1
}

func NewAnnSensus(disableConsensus bool, cryptoType crypto.CryptoType, campaign bool, partnerNum, threshold int,
	genesisAccounts []crypto.PublicKey, configFile string, disableTermChange bool) *AnnSensus {
	ann := &AnnSensus{}
	ann.disable = disableConsensus
	if len(genesisAccounts) < partnerNum && !disableConsensus {
		panic("need more account")
	}

	ann.close = make(chan struct{})
	ann.newTxHandlers = []chan types.Txi{}
	ann.campaignFlag = campaign
	ann.NbParticipants = partnerNum
	ann.Threshold = threshold
	ann.ConsensusTXConfirmed = make(chan []types.Txi)
	ann.cryptoType = cryptoType
	ann.dkgPulicKeyChan = make(chan kyber.Point)
	ann.UpdateEvent = make(chan bool)
	dkg := newDkg(ann, campaign, partnerNum, Maj23(partnerNum))
	ann.dkg = dkg
	ann.genesisAccounts = genesisAccounts
	t := newTerm(0, partnerNum)
	ann.term = t
	ann.newTermChan = make(chan bool)
	ann.genesisPkChan = make(chan *types.MessageConsensusDkgGenesisPublicKey)
	ann.NewPeerConnectedEventListener = make(chan string)

	ann.ProposalSeqChan = make(chan types.Hash)
	ann.NewLatestSequencer = make(chan bool)
	ann.startTermChange = make(chan bool)
	log.WithField("NbParticipants ", ann.NbParticipants).Info("new ann")

	ann.ConfigFilePath = configFile
	ann.termChangeChan = make(chan *types.TermChange)
	log.WithField("nbpartner ", ann.NbParticipants).Info("new ann")
	ann.disableTermChange = disableTermChange

	return ann
}

func (as *AnnSensus) InitAccount(myAccount *account.SampleAccount, sequencerTime time.Duration,
	judgeNonce func(me *account.SampleAccount) uint64, txCreator *og.TxCreator) {
	as.MyAccount = myAccount
	var myId int
	for id, pk := range as.genesisAccounts {
		if bytes.Equal(pk.Bytes, as.MyAccount.PublicKey.Bytes) {
			as.isGenesisPartner = true
			myId = id
			log.WithField("my id ", id).Info("i am a genesis partner")
		}
	}
	as.dkg.SetId(myId)
	as.bft = NewBFT(as, as.NbParticipants, myId, sequencerTime, judgeNonce, txCreator)
	as.addBftPartner()
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")
	if as.disable {
		log.Warn("annsensus disabled")
		return
	}
	as.dkg.start()
	as.bft.Start()
	goroutine.New(as.loop)
	goroutine.New(as.termChangeLoop)
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus will Stop")
	if as.disable {
		return
	}
	as.bft.Stop()
	as.dkg.stop()
	close(as.close)
}

func (as *AnnSensus) Name() string {
	return "AnnSensus"
}

func (as *AnnSensus) GetBenchmarks() map[string]interface{} {
	// TODO
	return nil
}

func (as *AnnSensus) GetCandidate(addr types.Address) *types.Campaign {
	return as.term.GetCandidate(addr)
}

func (as *AnnSensus) Candidates() map[types.Address]*types.Campaign {
	return as.term.Candidates()
}

func (as *AnnSensus) GetAlsoran(addr types.Address) *types.Campaign {
	return as.term.GetAlsoran(addr)
}

func (as *AnnSensus) Alsorans() map[types.Address]*types.Campaign {
	return as.term.Alsorans()
}

// RegisterNewTxHandler add a channel into AnnSensus.newTxHandlers. These
// channels are responsible to process new txs produced by annsensus.
func (as *AnnSensus) RegisterNewTxHandler(c chan types.Txi) {
	as.newTxHandlers = append(as.newTxHandlers, c)
}

// ProdCampaignOn let annsensus start producing campaign.
func (as *AnnSensus) ProduceCampaignOn() {
	as.produceCampaign()
}

// campaign continuously generate camp tx until AnnSensus.CampaingnOff is called.
func (as *AnnSensus) produceCampaign() {
	//generate new dkg public key for every campaign
	if as.term.HasCampaign(as.MyAccount.Address) {
		log.WithField("als", as.term.alsorans).WithField("cps ", as.term.campaigns).WithField("candidates", as.term.candidates).Debug("has campaign ")
		return
	}
	candidatePublicKey := as.dkg.GenerateDkg()
	// generate campaign.
	camp := as.genCamp(candidatePublicKey)
	if camp == nil {
		log.Warn("gen camp fail")
		return
	}
	log.WithField("pk ", hexutil.Encode(candidatePublicKey[:5])).WithField("cp ", camp).Debug("gen campaign")
	// send camp
	goroutine.New(func() {
		for _, c := range as.newTxHandlers {
			c <- camp
		}
	})

}

// commit takes a list of campaigns as input and record these
// camps' information. It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) commit(camps []*types.Campaign) {

	for i, c := range camps {
		if as.isTermChanging() {
			// add those unsuccessful camps into alsoran list.
			log.Debug("is termchanging ")
			as.AddAlsorans(camps[i:])
			return
		}

		err := as.AddCampaign(c)
		if err != nil {
			log.WithError(err).Debug("add campaign err")
			//continue
		}
		if !as.canChangeTerm() {
			continue
		}
		// start term changing.
		as.term.SwitchFlag(true)
		log.Debug("will term Change")
		as.startTermChange <- true

	}
}

// AddCandidate adds campaign into annsensus if the campaign meets the
// candidate requirements.
func (as *AnnSensus) AddCampaign(cp *types.Campaign) error {

	if as.term.HasCampaign(cp.Issuer) {
		log.WithField("id ", as.dkg.GetId()).WithField("campaign", cp).Debug("duplicate campaign ")
		return fmt.Errorf("duplicate ")
	}

	pubkey := cp.GetDkgPublicKey()
	if pubkey == nil {
		log.WithField("nil PartPubf for  campain", cp).Warn("add campaign")
		return fmt.Errorf("pubkey is nil ")
	}

	//as.dkg.AddPartner(cp, &as.MyAccount.PublicKey)
	as.term.AddCampaign(cp)
	log.WithField("me ", as.dkg.GetId()).WithField("add cp", cp).Debug("added")
	return nil
}

// AddAlsorans adds a list of campaigns into annsensus as alsorans.
// Campaigns will be regard as alsoran when current candidates cached
// already reaches the term change requirements.
func (as *AnnSensus) AddAlsorans(cps []*types.Campaign) {
	as.term.AddAlsorans(cps)
}

// isTermChanging checks if the annsensus system is currently
// changing its senators term.
func (as *AnnSensus) isTermChanging() bool {
	return as.term.Changing()
}

// canChangeTerm returns true if the candidates cached reaches the
// term change requirments.
func (as *AnnSensus) canChangeTerm() bool {
	ok := as.term.CanChange(as.Idag.LatestSequencer().Height, atomic.LoadUint32(&as.genesisBftIsRunning) == 1)
	return ok
}

func (as *AnnSensus) termChangeLoop() {
	// start term change gossip

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus termchange stopped")
			return

		case <-as.startTermChange:
			log := as.dkg.log()
			cp := as.term.GetCampaign(as.MyAccount.Address)
			if cp ==nil {
				log.Warn("cannot found campaign, i ma not a partner , no dkg gossip")
			}
			as.dkg.Reset(cp)
			as.dkg.SelectCandidates()
			if !as.dkg.isValidPartner {
				log.Debug("i am not a lucky dkg partner quit")
				as.term.SwitchFlag(false)
				continue
			}
			log.Debug("start dkg gossip")
			goroutine.New(as.dkg.StartGossip)

		case pk := <-as.dkgPulicKeyChan:
			log := as.dkg.log()
			log.WithField("pk ", pk).Info("got a bls public key from dkg")
			//after dkg  gossip finished, set term change flag
			//term change is finished
			as.term.SwitchFlag(false)
			sigset := as.dkg.GetBlsSigsets()
			log.WithField("sig sets ", sigset).Info("got sigsets ")
			//continue //for test case commit this
			tc := as.genTermChg(pk, sigset)
			if tc == nil {
				log.Error("tc is nil")
				continue
			}
			if atomic.CompareAndSwapUint32(&as.genesisBftIsRunning, 1, 0) {
				as.term.ChangeTerm(tc, as.Idag.GetHeight())
				goroutine.New(func() {
					as.newTermChan <- true
					//as.newTermChan <- struct{}{}
					//as.ConsensusTXConfirmed <- types.Txis{tc}
				})
				atomic.StoreUint32(&as.genesisBftIsRunning, 0)

				continue
			}
			//save the tc and sent it in next sequencer
			//may be it is a solution for the bug :
			//while syncing with bloom filter
			as.mu.RLock()
			as.currentTermChange = tc
			as.mu.RUnlock()
		}
	}

}

// pickTermChg picks a valid TermChange from a tc list.
func (as *AnnSensus) pickTermChg(tcs []*types.TermChange) (*types.TermChange, error) {
	if len(tcs) == 0 {
		return nil, errors.New("nil tcs")
	}
	var niceTc *types.TermChange
	for _, tc := range tcs {
		if niceTc != nil && niceTc.IsSameTermInfo(tc) {
			continue
		}
		if tc.TermID == as.term.ID()+1 {
			niceTc = tc
		}
	}
	if niceTc == nil {
		log.WithField("tc term id ", tcs[0].TermID).WithField("term id ", as.term.ID()).Warn("pic fail")
		return nil, errors.New("not found")
	}
	return niceTc, nil
}

//genTermChg
func (as *AnnSensus) genTermChg(pk kyber.Point, sigset []*types.SigSet) *types.TermChange {
	base := types.TxBase{
		Type: types.TxBaseTypeTermChange,
	}

	pkbls, err := pk.MarshalBinary()
	if err != nil {
		return nil
	}

	tc := &types.TermChange{
		TxBase: base,
		TermID: as.term.ID() + 1,
		Issuer: as.MyAccount.Address,
		PkBls:  pkbls,
		SigSet: sigset,
	}
	return tc
}

func (as *AnnSensus) addGenesisCampaigns() {
	as.mu.RLock()
	if  as.addedGenesisCampaign {
		as.mu.RUnlock()
		log.Info("already added genesis campaign")
		return
	}
	as.addedGenesisCampaign = true
	as.mu.RUnlock()
		for _, pk := range as.genesisAccounts {
			cp := types.Campaign{
				Issuer: pk.Address(),
			}
			as.term.AddCandidate(&cp, pk)
		}
}

func (as *AnnSensus) addBftPartner() {
	log.Debug("add bft partners")
	//should participate in genesis  bft process
	if !as.dkg.dkgOn {
		as.dkg.dkgOn = true
	}

	var peers []BFTPartner
	for i, pk := range as.genesisAccounts {
		//the third param is not used in peer
		peers = append(peers, NewOgBftPeer(pk, as.NbParticipants, i, time.Second))
	}
	as.bft.BFTPartner.SetPeers(peers)
}

func (as *AnnSensus) loop() {
	//var camp bool

	// sequencer entry
	var genesisCamps []*types.Campaign
	var peerNum int
	var initDone bool

	var lastHeight uint64
	var termId uint64
	var genesisPublickeyProcessFinished bool
	var sentCampaign uint64

	var eventInit bool
	//var lastheight uint64

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus loop stopped")
			return

		//TODO sequencer generate a random seed ,use random seed to select candidate peers
		case txs := <-as.ConsensusTXConfirmed:
			log.WithField(" txs ", txs).Debug("got consensus txs")
			if as.disableTermChange {
				log.Debug("term change is disabled , quiting")
				continue
			}
			var cps []*types.Campaign
			var tcs []*types.TermChange
			for _, tx := range txs {
				if tx.GetType() == types.TxBaseTypeCampaign {
					cp := tx.(*types.Campaign)
					cps = append(cps, cp)
					if bytes.Equal(cp.Issuer.Bytes[:], as.MyAccount.Address.Bytes[:]) {
						if sentCampaign > 0 {
							log.Debug("i sent a campaign")
							sentCampaign = 0
						}
					}
				} else if tx.GetType() == types.TxBaseTypeTermChange {
					tcs = append(tcs, tx.(*types.TermChange))
				}
			}
			// TODO:
			// here exists a bug:
			// the isTermChanging check should be here not in commit()

			if len(tcs) > 0 {
				tc, err := as.pickTermChg(tcs)
				if err != nil {
					log.Errorf("the received termChanges are not correct.")
					goto HandleCampaign
				}
				if as.isTermChanging() {
					log.Debug("is term changing")
					goto HandleCampaign
				}
				err = as.term.ChangeTerm(tc, as.Idag.LatestSequencer().Height)
				if err != nil {
					log.Errorf("change term error: %v", err)
					goto HandleCampaign
				}
				//if i am not a dkg partner , dkg publick key will got from termChange tx
				if !as.dkg.isValidPartner {
					pk, err := bn256.UnmarshalBinaryPointG2(tc.PkBls)
					if err != nil {
						log.WithError(err).Warn("unmarshal failed dkg joint public key")
						continue
					}
					as.dkg.partner.jointPubKey = pk
				}
				//send the signal if i am a partner of consensus nodes
				//if as.dkg.isValidPartner {
					goroutine.New(func() {
						as.newTermChan <- true
					})
				//} else {
				//	log.Debug("is not a valid partner")
				//}

			}
		HandleCampaign:
			//handle term change should be after handling campaign
			if len(cps) > 0 {
				as.commit(cps)
				log.Infof("already candidates: %d, alsorans: %d", len(as.Candidates()), len(as.Alsorans()))
			}

		case <-as.newTermChan:
			// sequencer generate entry.

			// TODO:
			// 1. check if yourself is the miner.
			// 2. produce raw_seq and broadcast it to network.
			// 3. start bft until someone produce a seq with BLS sig.
			log.Info("got newTermChange signal")
			as.bft.Reset(as.dkg.TermId, as.term.formerPublicKeys, int(as.dkg.partner.Id))
			// 3. start pbft until someone produce a seq with BLS sig.
			//TODO how to save consensus data
			if as.dkg.TermId == 1 {
			}

			if as.dkg.isValidPartner {
				as.SaveConsensusData()
				as.bft.startBftChan <- true
			}else {
				log.Debug("is not a valid partner")
			}


		//
		//case <-time.After(time.Millisecond * 300):
		//	height := as.Idag.LatestSequencer().Height
		//	if height <= lastheight {
		//		//should updated
		//		continue
		//	}
		//	lastheight = height
		//	if height > 2 && height%15 == 0 {
		//		//may i send campaign ?
		//		as.ProdCampaignOn()
		//	}

		case <-as.NewLatestSequencer:
			var tc *types.TermChange
			as.mu.RLock()
			tc = as.currentTermChange
			as.currentTermChange = nil
			as.mu.RUnlock()
			if tc != nil {
				for _, c := range as.newTxHandlers {
					c <- tc
				}
			}

			if !as.isTermChanging() {
				if as.canChangeTerm() {
					// start term changing.
					as.term.SwitchFlag(true)
					log.Debug("will termChange")
					as.startTermChange <- true
				}
			}

			if !as.campaignFlag || as.disableTermChange {
				continue
			}
			height := as.Idag.LatestSequencer().Height
			if height <= lastHeight {
				//should updated
				continue
			}
			if sentCampaign < height+2 && sentCampaign > 0 {
				//we sent campaign but did't receive them
				//sent again
				log.WithField("in term ", as.term.ID()).Debug("will generate campaign")
				as.ProduceCampaignOn()
				sentCampaign = height
			}
			//if term id is updated , we can produce campaign tx
			lastHeight = height
			if termId < as.term.ID() {
				termId = as.term.ID()
				//may i send campaign ?
				if sentCampaign == height {
					continue
				}
				log.WithField("in term ", as.term.ID()).Debug("will generate campaign in new term")
				as.ProduceCampaignOn()
				sentCampaign = height
			}

		case isUptoDate := <-as.UpdateEvent:
			as.TxEnable = isUptoDate
			height := as.Idag.LatestSequencer().Height
			log.WithField("height ", height).WithField("v ", isUptoDate).Info("get isUptoDate event")
			if eventInit {
				continue
			}

			if as.isGenesisPartner {
				//participate in bft process
				if height == 0 && isUptoDate {
					atomic.StoreUint32(&as.genesisBftIsRunning, 1)
					//wait until connect enought peers
					eventInit = true
					continue
				}
				//in this case  , node is a genesis partner, load data
				if as.dkg.partner.jointPubKey == nil && height == 0 {
					//load consensus data
					config, err := as.LoadConsensusData()
					if err != nil {
						log.WithError(err).Error("load error")
						panic(err)
					}
					as.SetConfig(config)
					log.Debug("will set config")
					as.addGenesisCampaigns()
					tc := as.genTermChg(as.dkg.partner.jointPubKey, nil)
					as.term.ChangeTerm(tc, height)
				}
				if isUptoDate {
					//after updated , start bft process, skip bfd gossip
					log.Debug("start bft")
					eventInit = true
					//TODO  newTermChange if we got new Termchange
					//goroutine.New (func() {
					//	as.newTermChan <- true
					//})

				}
			} else {
				//not a consensus partner , obtain genesis dkg public key from network
				//
				if as.term.started {
					eventInit = true
					continue
				}
				msg := types.MessageTermChangeRequest{
					Id: og.MsgCounter.Get(),
				}
				as.Hub.BroadcastMessage(og.MessageTypeTermChangeRequest, &msg)

			}

		case <-as.NewPeerConnectedEventListener:

			if !as.isGenesisPartner && !eventInit {
				msg := types.MessageTermChangeRequest{
					Id: og.MsgCounter.Get(),
				}
				as.Hub.BroadcastMessage(og.MessageTypeTermChangeRequest, &msg)
			}
			if initDone {
				continue
			}
			peerNum++
			log.WithField("num ", peerNum).Debug("peer num ")
			if peerNum == as.NbParticipants-1 && atomic.LoadUint32(&as.genesisBftIsRunning) == 1 {
				log.Info("start dkg genesis consensus")
				goroutine.New(func() {
					as.dkg.SendGenesisPublicKey()
				})
				initDone = true
			}

			//for genesis consensus , obtain tc from network
		case tc := <-as.termChangeChan:
			if eventInit {
				log.WithField("tc ",tc).Warn("already handle genesis tc")
			}
			pk, err := bn256.UnmarshalBinaryPointG2(tc.PkBls)
			if err != nil {
				log.WithError(err).Warn("unmarshal failed dkg joint public key")
				continue
			}
			as.dkg.partner.jointPubKey = pk
			eventInit = true
			as.addGenesisCampaigns()
			as.term.ChangeTerm(tc, as.Idag.GetHeight())

			//
		case pkMsg := <-as.genesisPkChan:
			if genesisPublickeyProcessFinished {
				log.WithField("pkMsg ", pkMsg).Warn("already finished, don't send again")
				continue
			}
			id := -1
			for i, pk := range as.genesisAccounts {
				if bytes.Equal(pkMsg.PublicKey, pk.Bytes) {
					id = i
					break
				}
			}
			if id < 0 {
				log.Warn("not valid genesis pk")
				continue
			}
			cp := &types.Campaign{
				DkgPublicKey: pkMsg.DkgPublicKey,
				Issuer:       as.genesisAccounts[id].Address(),
				TxBase: types.TxBase{
					PublicKey: pkMsg.PublicKey,
					Weight:    uint64(id*10 + 10),
				},
			}
			err := cp.UnmarshalDkgKey(bn256.UnmarshalBinaryPointG2)
			if err != nil {
				log.WithField("pk data ", hexutil.Encode( cp.DkgPublicKey)).WithError(err).Warn("unmarshal error")
				continue
			}

			as.mu.RLock()
			genesisCamps = append(genesisCamps, cp)
			as.mu.RUnlock()
			log.WithField("id ", id).Debug("got geneis pk")
			if len(genesisCamps) == as.NbParticipants {
				goroutine.New(func() {
					as.commit(genesisCamps)
				})
				genesisPublickeyProcessFinished = true
			}

		case hash := <-as.ProposalSeqChan:
			log.WithField("hash ", hash.TerminalString()).Debug("got proposal seq hash")
			request := as.bft.proposalCache[hash]
			if request != nil {
				delete(as.bft.proposalCache, hash)
				m := Message{
					Type:    og.MessageTypeProposal,
					Payload: request,
				}
				as.bft.BFTPartner.GetIncomingMessageChannel() <- m

			}

		}
	}

}

type DKGInfo struct {
	TermId             int                   `json:"term_id"`
	Id                 uint32                `json:"id"`
	PartPubs           []kyber.Point         `json:"part_pubs"`
	MyPartSec          kyber.Scalar          `json:"-"`
	CandidatePartSec   []kyber.Scalar        `json:"-"`
	CandidatePublicKey [][]byte              `json:"candidate_public_key"`
	AddressIndex       map[types.Address]int `json:"address_index"`
}

type BFTInfo struct {
	BFTPartner    PeerInfo      `json:"bft_partner"`
	DKGTermId     int           `json:"dkg_term_id"`
	SequencerTime time.Duration `json:"sequencer_time"`
	Partners      []PeerInfo    `json:"partners"`
}

func (a *AnnSensus) GetDkgBftInfo() (*DKGInfo, *BFTInfo) {
	dkgInfo := DKGInfo{
		TermId:             a.dkg.TermId,
		Id:                 a.dkg.partner.Id,
		PartPubs:           a.dkg.partner.PartPubs,
		MyPartSec:          a.dkg.partner.MyPartSec,
		CandidatePublicKey: a.dkg.partner.CandidatePublicKey,
		CandidatePartSec:   a.dkg.partner.CandidatePartSec,
		AddressIndex:       a.dkg.partner.addressIndex,
	}

	bftInfo := BFTInfo{
		BFTPartner:    a.bft.BFTPartner.PeerInfo,
		Partners:      a.bft.BFTPartner.PeersInfo,
		DKGTermId:     a.bft.DKGTermId,
		SequencerTime: a.bft.SequencerTime,
	}
	return &dkgInfo, &bftInfo
}
