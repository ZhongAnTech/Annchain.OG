package annsensus

import (
	"sync"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type AnnSensus struct {
	cryptoType crypto.CryptoType
	doCamp     bool // the switch of whether annsensus should produce campaign.

	campaignFlag bool
	maxCamps     int
	candidates   map[types.Address]*types.Campaign
	alsorans     map[types.Address]*types.Campaign
	//campaigns    map[types.Address]*types.Campaign // TODO replaced by candidates

	termchgFlag bool

	// channels to send txs.
	newTxHandlers []chan types.Txi

	//receive consensus txs from pool ,for notifications
	ConsensusTXConfirmed chan []types.Txi

	dkg *Dkg

	// signal channels
	termChgStartSignal chan struct{}
	termChgEndSignal   chan []*types.TermChange
	dkgPkCh            chan kyber.Point
	dkgReqCh           chan *types.MessageConsensusDkgDeal
	dkgRespCh          chan *types.MessageConsensusDkgDealResponse

	Hub            MessageSender // todo use interface later
	Txpool         og.ITxPool
	Idag           og.IDag
	partner        *Partner // partner is for distributed key generate.
	MyPrivKey      *crypto.PrivateKey
	Threshold      int
	NbParticipants int

	mu          sync.RWMutex
	termchgLock sync.RWMutex
	close       chan struct{}
}

func NewAnnSensus(cryptoType crypto.CryptoType, campaign bool, partnerNum, threshold int) *AnnSensus {
	ann := &AnnSensus{}

	ann.close = make(chan struct{})
	ann.newTxHandlers = []chan types.Txi{}
	ann.termChgStartSignal = make(chan struct{})
	ann.termChgEndSignal = make(chan []*types.TermChange)
	ann.campaignFlag = campaign
	ann.candidates = make(map[types.Address]*types.Campaign)
	ann.alsorans = make(map[types.Address]*types.Campaign)
	ann.NbParticipants = partnerNum
	ann.Threshold = threshold
	ann.ConsensusTXConfirmed = make(chan []types.Txi)
	ann.cryptoType = cryptoType

	dkg := newDkg(ann, campaign, partnerNum, threshold)
	ann.dkg = dkg

	return ann
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")
	if as.campaignFlag {
		as.ProdCampaignOn()
		// TODO campaign gossip starts here?
		go as.gossipLoop()
	}
	go as.loop()
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus Stop")
	as.ProdCampaignOff()
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
	return as.candidates[addr]
}

func (as *AnnSensus) Candidates() map[types.Address]*types.Campaign {
	return as.candidates
}

// RegisterNewTxHandler add a channel into AnnSensus.newTxHandlers. These
// channels are responsible to process new txs produced by annsensus.
func (as *AnnSensus) RegisterNewTxHandler(c chan types.Txi) {
	as.newTxHandlers = append(as.newTxHandlers, c)
}

// ProdCampaignOn let annsensus start producing campaign.
func (as *AnnSensus) ProdCampaignOn() {
	as.doCamp = true
	go as.prodcampaign()
}

// ProdCampaignOff let annsensus stop producing campaign.
func (as *AnnSensus) ProdCampaignOff() {
	as.doCamp = false
}

// campaign continuously generate camp tx until AnnSensus.CampaingnOff is called.
func (as *AnnSensus) prodcampaign() {
	// TODO
	for {
		select {
		case <-as.close:
			log.Info("campaign stopped due to annsensus closed")
			return
		case <-time.After(time.Second * 4):
			if !as.doCamp {
				log.Info("campaign stopped")
				return
			}
			// generate dkg partner and key pair.

			// delete this later
			pubKey := as.GenerateDkg()

			as.dkg.GenerateDkg()
			pubKey = as.dkg.PublicKey()

			// generate campaign.
			camp := as.genCamp(pubKey)
			// send camp
			if camp != nil {
				for _, c := range as.newTxHandlers {
					c <- camp
				}
			}

			// as.doCamp = false
		}
	}
}

// commit takes a list of campaigns as input and record these
// camps' information It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) commit(camps []*types.Campaign) {
	// TODO

	for i, c := range camps {
		if as.isTermChanging() {
			// add those unsuccessful camps into alsoran list.
			log.Debug("is termchanging ")
			as.addAlsorans(camps[i:])
			return
		}
		// TODO
		// handle campaigns should not only add it into candidate list.
		// as.candidates[c.Issuer] = c
		as.AddCampaignCandidates(c) //todo remove duplication here
		if !as.canChangeTerm() {
			continue
		}
		as.SwitchTcFlagWithLock(true)

		camps := []*types.Campaign{}
		for _, camp := range as.candidates {
			camps = append(camps, camp)
		}
		as.changeTerm(camps)

	}

}

// canChangeTerm returns true if the campaigns cached reaches the
// term change requirments.
func (as *AnnSensus) canChangeTerm() bool {
	// TODO

	if len(as.candidates) == 0 {
		return false
	}
	if len(as.candidates) < as.NbParticipants {
		log.WithField("len ", len(as.candidates)).Debug("not enough campaigns , waiting")
		return false
	}

	return true
}

func (as *AnnSensus) isTermChanging() bool {
	as.termchgLock.RLock()
	changing := as.termchgFlag
	as.termchgLock.RUnlock()

	return changing
}

func (as *AnnSensus) changeTerm(camps []*types.Campaign) {
	// start term change gossip
	as.dkg.loadCampaigns(camps)
	as.termChgStartSignal <- struct{}{}

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus termchansge stopped")
			return

		case pk := <-as.dkgPkCh:
			log.Info("got a bls public key from dkg: %s", pk.String())
			tc := as.genTermChg(pk)
			if tc != nil {
				for _, c := range as.newTxHandlers {
					c <- tc
				}
			}

			// TODO
			// temporarily clear the candidates and alsorans, because there
			// is no TermChange produced now.
			as.candidates = make(map[types.Address]*types.Campaign)
			as.alsorans = make(map[types.Address]*types.Campaign)

		case tcs := <-as.termChgEndSignal:
			// TODO
			// handle TermChanges
			if len(tcs) > 1 {
				// TODO
			}

		}
	}

}

func (as *AnnSensus) genTermChg(pk kyber.Point) *types.TermChange {
	// TODO

	return nil
}

func (as *AnnSensus) SwitchTcFlagWithLock(flag bool) {
	as.termchgLock.Lock()
	defer as.termchgLock.Unlock()

	as.termchgFlag = flag
}

// addAlsorans add a list of campaigns into alsoran list.
func (as *AnnSensus) addAlsorans(camps []*types.Campaign) {
	// TODO
	for _, cp := range camps {
		if cp == nil {
			continue
		}
		if as.HasCampaign(cp) {
			continue
		}
		as.alsorans[cp.Issuer] = cp
	}
}

func (as *AnnSensus) loop() {

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus loop stopped")
			return

		// case camps := <-as.campsCh:
		// 	fmt.Println(camps)
		// 	// TODO
		// 	// case commit

		// case termchg := <-as.termchgCh:
		// 	fmt.Println(termchg)
		// 	// TODO
		// 	// case start term change gossip
		// 	// dag sent campaigns and termchanges tx

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
				log.Infof("already candidates: %d, alsorans: %d", len(as.candidates), len(as.alsorans))
			}

			if len(tcs) > 0 {

			}

		}
	}

}
