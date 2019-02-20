package annsensus

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/poc/dkg"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"time"
)

type AnnSensus struct {
	doCamp bool // the switch of whether annsensus should produce campaign.

	receivers []chan types.Txi

	close chan struct{}
	campaignFlag  bool
	MyPrivKey  *crypto.PrivateKey
	campaigns  map[types.Address]*types.Campaign //temperary

	Hub  *og.Hub     //todo use interface later
	Txpool og.ITxPool
	Idag   og.IDag
	partNer *dkg.Partner
	Threshold             int
	NbParticipants        int
 }

func NewAnnSensus(campaign bool ) *AnnSensus {
	return &AnnSensus{
		close:     make(chan struct{}),
		receivers: []chan types.Txi{},
		campaignFlag :campaign,
		campaigns: make(map[types.Address]*types.Campaign),
	}
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")
	if as.campaignFlag{
		as.CampaignOn()
		go as.gossipLoop()
	}
	// TODO
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus Stop")
	as.CampaignOff()
	close(as.close)
}

func (as *AnnSensus) Name() string {
	return "AnnSensus"
}

func (as *AnnSensus) GetBenchmarks() map[string]interface{} {
	// TODO
	return nil
}

// RegisterReceiver add a channel into AnnSensus.receivers, receivers are the
// hooks that handle new txs.
func (as *AnnSensus) RegisterReceiver(c chan types.Txi) {
	as.receivers = append(as.receivers, c)
}

func (as *AnnSensus) CampaignOn() {
	as.doCamp = true
	go as.campaign()
}

func (as *AnnSensus) CampaignOff() {
	as.doCamp = false
}

func (as *AnnSensus) campaign() {
	// TODO
	for {
		select {
        case <-as.close:
			log.Info("campaign stopped")
			return
        case <-time.After(time.Second):
		if !as.doCamp {
			log.Info("campaign stopped")
			return
		}
		// generate campaign.
		camp := as.genCamp()

		// send camp
		if camp!=nil {
			for _, c := range as.receivers {
				c <- camp
			}
		}

		}

	}
}


// commit takes a list of campaigns as input and record these
// camps' information It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) commit(camps []*types.Campaign) {

}
