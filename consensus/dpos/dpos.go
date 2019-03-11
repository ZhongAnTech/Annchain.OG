package dpos

import (
	"github.com/annchain/OG/consensus"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

// Dpos Delegate Proof-of-Stake
type Dpos struct {
	dag      consensus.IDag
	quitCh   chan bool
	proposer *types.Address
}

type ConsensusState uint64

func NewDpos(dag consensus.IDag, addr *types.Address) *Dpos {
	return &Dpos{
		dag:      dag,
		quitCh:   make(chan bool),
		proposer: addr,
	}
}

// Start start dpos service.
func (d *Dpos) Start() {
	log.Info("Starting Dpos ...")
	go d.loop()
}

// Stop stop dpos service.
func (d *Dpos) Stop() {
	log.Info("Stopping Dpos ...")
	d.quitCh <- true
}

func (d *Dpos) Name() string {
	return "dpos"
}

func (d *Dpos) loop() {
	for {
		select {
		case <-d.quitCh:
			log.Info("dpos received quit message. Quitting...")
			return
		}
	}
}

//VerifySequencer verify received sequencer
func (d *Dpos) VerifySequencer(seq *types.Sequencer) error {
	//todo
	return nil
}

//newSequncer create new sequencer if it is our time
func (d *Dpos) newSequncer() (seq *types.Sequencer) {
	//todo
	return nil
}

//checkDeadLine  check whether the sequencer proposed  in valid time
func (d *Dpos) checkDeadLine(seq *types.Sequencer) error {
	return nil
}

//
func (d *Dpos) checkProposer(seq *types.Sequencer) (state ConsensusState, err error) {
	//todo
	return 0, nil
}
