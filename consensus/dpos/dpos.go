package dpos

import (
	"github.com/annchain/OG/consensus"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

// Dpos Delegate Proof-of-Stake
type Dpos struct {
        dag  consensus.IDag
	    quitCh chan bool
        Issuer  *types.Address
}

func NewDpos ( dag consensus.IDag, addr * types.Address) *Dpos {
	return &Dpos{
		dag :dag,
		quitCh:make(chan bool ),
		Issuer:addr,
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

func ( d *Dpos)Name ()string {
	return "dpos"
}

func (d *Dpos)loop () {
	for {
		select {
		case <- d.quitCh:
			log.Info("dpos received quit message. Quitting...")
			return
		}
	}
}