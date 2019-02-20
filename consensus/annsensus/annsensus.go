package annsensus

import (
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type AnnSensus struct {
	doCamp bool // the switch of whether annsensus should produce campaign.

	newTxHandlers []chan types.Txi

	close chan struct{}
}

func NewAnnSensus() *AnnSensus {
	return &AnnSensus{
		close:         make(chan struct{}),
		newTxHandlers: []chan types.Txi{},
	}
}

func (as *AnnSensus) Start() {
	log.Info("AnnSensus Start")
	// TODO
}

func (as *AnnSensus) Stop() {
	log.Info("AnnSensus Stop")

	close(as.close)
}

func (as *AnnSensus) Name() string {
	return "AnnSensus"
}

func (as *AnnSensus) GetBenchmarks() map[string]interface{} {
	// TODO
	return nil
}

// RegisterReceiver add a channel into AnnSensus.newTxHandlers, newTxHandlers
// are the hooks that handle new txs.
func (as *AnnSensus) RegisterReceiver(c chan types.Txi) {
	as.newTxHandlers = append(as.newTxHandlers, c)
}

func (as *AnnSensus) CampaignOn() {
	as.doCamp = true
	go as.campaign()
}

func (as *AnnSensus) CampaignOff() {
	as.doCamp = false
}

// campaign continuously generate camp tx until AnnSensus.CampaingnOff is called.
func (as *AnnSensus) campaign() {
	// TODO
	for {
		if !as.doCamp {
			return
		}
		// generate campaign.
		camp := as.genCamp()

		// send camp
		for _, c := range as.newTxHandlers {
			c <- camp
		}

	}
}

// genCamp calculate vrf and generate a campaign that contains this vrf info.
func (as *AnnSensus) genCamp() *types.Campaign {
	// TODO
	return &types.Campaign{}
}

// commit takes a list of campaigns as input and record these
// camps' information It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) commit(camps []*types.Campaign) {

}
