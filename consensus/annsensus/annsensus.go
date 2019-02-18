package annsensus

import (
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type AnnSensus struct {
	quitCh        chan bool
}

// Commit takes a list of campaigns as input and record these
// camps' information It checks if the number of camps reaches
// the threshold. If so, start term changing flow.
func (as *AnnSensus) Commit(camps []*types.Campaign) {

}

// func (as *AnnSensus)

func NewAnnSensus ()*AnnSensus {
	return &AnnSensus{
		quitCh:make(chan bool),
	}
}


func (a *AnnSensus)GetBenchmarks() map[string]interface{} {
	//todo
	return nil
}

// Start start dpos service.
func (a *AnnSensus) Start() {
	log.Info("Starting AnnSensus ...")
	go a.loop()
}

// Stop stop dpos service.
func (a *AnnSensus) Stop() {
	log.Info("Stopping AnnSensus ...")
	a.quitCh <- true
}

func (a *AnnSensus) Name() string {
	return "AnnSensus"
}

func (a *AnnSensus) loop() {
	for {
		select {
		case <-a.quitCh:
			log.Info("dpos received quit message. Quitting...")
			return
		}
	}
}