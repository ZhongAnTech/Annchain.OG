package annsensus

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type AnnSensus struct {
	cryptoType crypto.CryptoType

	dkg *Dkg

	campaignFlag bool
	maxCamps     int
	candidates   map[types.Address]*types.Campaign
	alsorans     map[types.Address]*types.Campaign
	// current senators to produce the sequencer.
	senators map[types.Address]*Senator

	termchgFlag        bool
	termChgStartSignal chan struct{}
	termChgEndSignal   chan *types.TermChange
	dkgPkCh            chan kyber.Point

	// channels to send txs.
	newTxHandlers []chan types.Txi
	//receive consensus txs from pool ,for notifications
	ConsensusTXConfirmed chan []types.Txi
	//LatestSequencerChan   chan bool

	// // channels for receiving txs.
	// campsCh   chan []*types.Campaign
	// termchgCh chan *types.TermChange

	// signal channels

	Hub    MessageSender // todo use interface later
	Txpool og.ITxPool
	Idag   og.IDag
	//partner        *Partner // partner is for distributed key generate.
	MyPrivKey      *crypto.PrivateKey
	Threshold      int
	NbParticipants int

	mu          sync.RWMutex
	termchgLock sync.RWMutex

	close chan struct{}
	id    int
}

func NewAnnSensus(cryptoType crypto.CryptoType, campaign bool, partnerNum, threshold int) *AnnSensus {
	ann := &AnnSensus{}

	ann.close = make(chan struct{})
	ann.newTxHandlers = []chan types.Txi{}
	ann.termChgStartSignal = make(chan struct{})
	ann.termChgEndSignal = make(chan *types.TermChange)
	ann.campaignFlag = campaign
	ann.candidates = make(map[types.Address]*types.Campaign)
	ann.alsorans = make(map[types.Address]*types.Campaign)
	ann.NbParticipants = partnerNum
	ann.Threshold = threshold
	ann.ConsensusTXConfirmed = make(chan []types.Txi)

	ann.cryptoType = cryptoType
	ann.dkgPkCh = make(chan kyber.Point)
	dkg := newDkg(ann, campaign, partnerNum, threshold)
	ann.dkg = dkg

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
	go as.prodcampaign()
}

// campaign continuously generate camp tx until AnnSensus.CampaingnOff is called.
func (as *AnnSensus) prodcampaign() {
	as.dkg.GenerateDkg()

	// generate campaign.
	camp := as.genCamp(as.dkg.PublicKey())
	// send camp
	if camp != nil {
		for _, c := range as.newTxHandlers {
			c <- camp
		}
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
			as.addAlsorans(camps[i:])
			return
		}
		// TODO
		// handle campaigns should not only add it into candidate list.
		// as.candidates[c.Issuer] = c
		err := as.AddCampaignCandidates(c) //todo remove duplication here
		if err != nil {
			log.WithError(err).Debug("add campaign err")
			continue
		}
		if !as.canChangeTerm() {
			continue
		}
		as.SwitchTcFlagWithLock(true)

		camps := []*types.Campaign{}
		for _, camp := range as.candidates {
			camps = append(camps, camp)
		}
		go as.changeTerm(camps)

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
	log := log.WithField("me", as.id)
	// start term change gossip
	as.dkg.loadCampaigns(camps)
	as.dkg.StartGossip()

	for {
		select {
		case <-as.close:
			log.Info("got quit signal , annsensus termchansge stopped")
			return

		case pk := <-as.dkgPkCh:
			log.Info("got a bls public key from dkg: %s", pk.String())

			sigs := as.dkg.blsSigSets
			log.WithField("sig sets ", sigs).Info("got sigsets ")
			continue //for test
			// TODO generate sigset in dkg gossip.
			sigset := map[types.Address][]byte{}
			tc := as.genTermChg(pk, sigset)
			if tc != nil {
				for _, c := range as.newTxHandlers {
					c <- tc
				}
			}

		}
	}

}

func (as *AnnSensus) ProcessTermChange(tc *types.TermChange) {
	// TODO
	// lock?

	as.senators = make(map[types.Address]*Senator)
	for addr, camp := range as.candidates {
		s := newSenator(addr, camp.PublicKey, tc.PkBls)
		as.senators[addr] = s
	}

	as.candidates = make(map[types.Address]*types.Campaign)
	as.alsorans = make(map[types.Address]*types.Campaign)

	as.SwitchTcFlagWithLock(false)
}

func (as *AnnSensus) genTermChg(pk kyber.Point, sigset map[types.Address][]byte) *types.TermChange {
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

func (as *AnnSensus) SwitchTcFlagWithLock(flag bool) {
	as.termchgLock.Lock()
	defer as.termchgLock.Unlock()

	as.termchgFlag = flag
}

// addAlsorans add a list of campaigns into alsoran list.
func (as *AnnSensus) addAlsorans(camps []*types.Campaign) {
	// TODO
	log.WithField("add also runs", camps).Trace(camps)
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
				log.Infof("already candidates: %d, alsorans: %d", len(as.candidates), len(as.alsorans))
			}

			if len(tcs) > 0 {
				//TODO
				var err error
				var tc *types.TermChange
				//tc, err := as.VerifyTermChanges(tcs)
				if err != nil {
					log.Errorf("the received termchanges are not correct.")
				}
				if !as.isTermChanging() {
					continue
				}
				as.ProcessTermChange(tc)
			}
		case <-time.After(time.Second * 6):
			if !camp {
				camp = false
				as.ProdCampaignOn()
			}

		}
	}

}
