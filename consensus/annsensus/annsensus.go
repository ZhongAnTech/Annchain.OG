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
	"github.com/annchain/OG/account"
	"sync"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type AnnSensus struct {
	id           int
	campaignFlag bool
	cryptoType   crypto.CryptoType

	dkg  *Dkg
	term *Term
	pbft *PBFT

	dkgPkCh              chan kyber.Point // channel for receiving dkg response.
	newTxHandlers        []chan types.Txi // channels to send txs.
	ConsensusTXConfirmed chan []types.Txi // receiving consensus txs from dag confirm.

	Hub    MessageSender // todo use interface later
	Txpool og.ITxPool
	Idag   og.IDag

	MyAccount      *account.SampleAccount
	Threshold      int
	NbParticipants int

	mu sync.RWMutex

	close            chan struct{}
	genesisAccounts  []crypto.PublicKey
	isGenesisPartner bool
	genesisBftIsRunning uint32
	UpdateEvent chan bool // syner update event
	newtermCh chan struct{}
}

func NewAnnSensus(cryptoType crypto.CryptoType, campaign bool, partnerNum, threshold int, sequencerTime time.Duration,
	judgeNonce func(me *account.SampleAccount) uint64, txcreator *og.TxCreator,
	Verifiers []og.Verifier, genesisAccounts []crypto.PublicKey) *AnnSensus {
	if len(genesisAccounts) < partnerNum {
		panic("need more account")
	}
	var myId int
	ann := &AnnSensus{}
	ann.close = make(chan struct{})
	ann.newTxHandlers = []chan types.Txi{}
	ann.campaignFlag = campaign
	ann.NbParticipants = partnerNum
	ann.Threshold = threshold
	ann.ConsensusTXConfirmed = make(chan []types.Txi)
	ann.cryptoType = cryptoType
	ann.dkgPkCh = make(chan kyber.Point)
	ann.UpdateEvent = make(chan bool)
	dkg := newDkg(ann, campaign, partnerNum, threshold)
	ann.dkg = dkg
	ann.genesisAccounts = genesisAccounts
	for id, pk := range ann.genesisAccounts {
		if bytes.Equal(pk.Bytes, ann.MyAccount.PublicKey.Bytes) {
			log.WithField("my id ", id).Info("i am a genesis partner")
			myId = id
		}
	}
	t := newTerm(0, partnerNum)
	ann.term = t
	ann.pbft = NewPBFT(partnerNum, myId, sequencerTime, judgeNonce, txcreator, Verifiers)
	ann.newtermCh = make(chan struct{})

	return ann
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")

	as.dkg.start()
	as.pbft.Start()
	go as.loop()
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus Stop")
	as.pbft.Stop()
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
func (as *AnnSensus) ProdCampaignOn() {
	go as.prodcampaign()
}

// campaign continuously generate camp tx until AnnSensus.CampaingnOff is called.
func (as *AnnSensus) prodcampaign() {
	as.dkg.GenerateDkg()

	// generate campaign.
	camp := as.genCamp(as.dkg.PublicKey())
	if camp == nil {
		return
	}
	// send camp
	for _, c := range as.newTxHandlers {
		c <- camp
	}

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

		err := as.AddCandidate(c)
		if err != nil {
			log.WithError(err).Debug("add campaign err")
			continue
		}
		if !as.canChangeTerm() {
			continue
		}
		// start term changing.
		as.term.SwitchFlag(true)
		camps := []*types.Campaign{}
		for _, camp := range as.Candidates() {
			camps = append(camps, camp)
		}
		go as.changeTerm(camps)

	}
}

// AddCandidate adds campaign into annsensus if the campaign meets the
// candidate requirements.
func (as *AnnSensus) AddCandidate(cp *types.Campaign) error {

	if as.term.HasCampaign(cp) {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
		return fmt.Errorf("duplicate ")
	}

	pubkey := cp.GetDkgPublicKey()
	if pubkey == nil {
		log.WithField("nil PartPubf for  campain", cp).Warn("add campaign")
		return fmt.Errorf("pubkey is nil ")
	}

	as.dkg.AddPartner(cp, &as.MyAccount.PublicKey)
	as.term.AddCandidate(cp)
	// log.WithField("me ",as.id).WithField("add cp", cp ).Debug("added")
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
	return as.term.CanChange()
}

func (as *AnnSensus) changeTerm(camps []*types.Campaign) {
	log := log.WithField("me", as.id)
	// start term change gossip
	as.dkg.loadCampaigns(camps)
	as.dkg.StartGossip()

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus termchange stopped")
			return

		case pk := <-as.dkgPkCh:
			log.WithField("pk ", pk).Info("got a bls public key from dkg")
			if  atomic.LoadUint32(&as.genesisBftIsRunning) ==1 {
				as.newtermCh<- struct{}{}
				continue
			}
			sigset := as.dkg.GetBlsSigsets()
			log.WithField("sig sets ", sigset).Info("got sigsets ")
			//continue //for test case commit this
			tc := as.genTermChg(pk, sigset)
			if tc == nil {
				continue
			}
			for _, c := range as.newTxHandlers {
				c <- tc
			}
		}
	}

}

// pickTermChg picks a valid TermChange from a tc list.
func (as *AnnSensus) pickTermChg(tcs []*types.TermChange) (*types.TermChange, error) {
	// TODO:
	// not implemented yet.

	return tcs[0], nil
}

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
		Issuer: as.MyAccount.Address,
		PkBls:  pkbls,
		SigSet: sigset,
	}
	return tc
}

func (as *AnnSensus) loop() {
	var camp bool

	// sequencer entry

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus loop stopped")
			return

		//TODO sequencer generate a random seed ,use random seed to select candidate peers
		case txs := <-as.ConsensusTXConfirmed:
			log.WithField(" txs ", txs).Debug("got consensus txs")
			var cps []*types.Campaign
			var tcs []*types.TermChange
			for _, tx := range txs {
				if tx.GetType() == types.TxBaseTypeCampaign {
					cps = append(cps, tx.(*types.Campaign))
				} else if tx.GetType() == types.TxBaseTypeTermChange {
					tcs = append(tcs, tx.(*types.TermChange))
				}
			}
			// TODO:
			// here exists a bug:
			// the isTermChanging check should be here not in commit()
			if len(cps) > 0 {
				as.commit(cps)
				log.Infof("already candidates: %d, alsorans: %d", len(as.Candidates()), len(as.Alsorans()))
			}
			if len(tcs) > 0 {
				tc, err := as.pickTermChg(tcs)
				if err != nil {
					log.Errorf("the received termchanges are not correct.")
				}
				if !as.isTermChanging() {
					continue
				}
				err = as.term.ChangeTerm(tc)
				if err != nil {
					log.Errorf("change term error: %v", err)
					continue
				}
				as.newtermCh <- struct{}{}
			}

		case <-as.newtermCh:
			// sequencer generate entry.

			// TODO:
			// 1. check if yourself is the miner.
			// 2. produce raw_seq and broadcast it to network.
			// 3. start pbft until someone produce a seq with BLS sig.
            log.Info("got newtermcahneg signal")
			as.pbft.startBftChan <- true

		case <-time.After(time.Second * 6):
			if !camp {
				camp = true
				as.ProdCampaignOn()
			}
		case isUptoDate :=<-as.UpdateEvent:
			log.WithField("v ",isUptoDate).Info("get isUptoDate event")
		    if isUptoDate && as.Idag.LatestSequencer().Height ==0 && as.isGenesisPartner {
		    	//should participate in genesis  bft process
		    	if !as.dkg.dkgOn {
		    		as.dkg.dkgOn = true
		    		atomic.StoreUint32(&as.genesisBftIsRunning,1)
		    		var peers []BFTPartner
		    		for i, pk := range as.genesisAccounts {
		    			if i==as.id{
		    				continue
						}
		    			//the third param is not used in peer
		    			peers = append(peers, NewOgBftPeer(pk,as.NbParticipants,i,time.Second))
					}
		    		as.pbft.BFTPartner.SetPeers(peers)
				}
			}
			//

		}
	}

}
