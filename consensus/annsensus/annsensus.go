package annsensus

import (
	"fmt"
	"sync"
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

	dkgPkCh              chan kyber.Point // channel for receiving dkg response.
	newTxHandlers        []chan types.Txi // channels to send txs.
	ConsensusTXConfirmed chan []types.Txi // receiving consensus txs from dag confirm.

	Hub    MessageSender // todo use interface later
	Txpool og.ITxPool
	Idag   og.IDag

	MyPrivKey      *crypto.PrivateKey
	Threshold      int
	NbParticipants int

	mu sync.RWMutex

	close chan struct{}
}

func NewAnnSensus(cryptoType crypto.CryptoType, campaign bool, partnerNum, threshold int) *AnnSensus {
	ann := &AnnSensus{}

	ann.close = make(chan struct{})
	ann.newTxHandlers = []chan types.Txi{}
	ann.campaignFlag = campaign
	ann.NbParticipants = partnerNum
	ann.Threshold = threshold
	ann.ConsensusTXConfirmed = make(chan []types.Txi)
	ann.cryptoType = cryptoType
	ann.dkgPkCh = make(chan kyber.Point)

	dkg := newDkg(ann, campaign, partnerNum, threshold)
	ann.dkg = dkg

	t := newTerm(0, partnerNum)
	ann.term = t

	return ann
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")

	as.dkg.start()
	go as.loop()
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus Stop")

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
// camps' information It checks if the number of camps reaches
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

	as.dkg.AddPartner(cp, as.MyPrivKey)
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

// canChangeTerm returns true if the campaigns cached reaches the
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
			log.Info("got a bls public key from dkg: %s", pk.String())

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
	// the term id should match.

	return nil, nil
}

func (as *AnnSensus) genTermChg(pk kyber.Point, sigset []*types.SigSet) *types.TermChange {
	base := types.TxBase{
		Type: types.TxBaseTypeTermChange,
	}
	address := as.MyPrivKey.PublicKey().Address()

	pkbls, err := pk.MarshalBinary()
	if err != nil {
		return nil
	}

	tc := &types.TermChange{
		TxBase: base,
		Issuer: address,
		PkBls:  pkbls,
		SigSet: sigset,
	}
	return tc
}

func (as *AnnSensus) loop() {
	var camp bool
	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus loop stopped")
			return

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
				as.term.ChangeTerm(tc)
			}

		case <-time.After(time.Second * 6):
			if !camp {
				camp = true
				as.ProdCampaignOn()
			}

		}
	}

}
