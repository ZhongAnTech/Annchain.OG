package dkg

import (
	"github.com/annchain/OG/common"
	dkg "github.com/annchain/kyber/v3/share/dkg/pedersen"
)

type Stage int

const (
	StageStart Stage = iota
	StageDealReceived
	//StageDealResponseReceived
	//StageReady
)

// DisorderedCache collects necessary prerequisites to make message in order
type DisorderedCache map[common.Address]Stagable

// Stagable will justify which stage it is currently on, depending on the messages it received
// e.g., if the messages received are 1,2,6,7,8, then the current stage is 2
type Stagable interface {
	GetCurrentStage() Stage
}

type DkgDiscussion struct {
	Deal *dkg.Deal
}

func (d *DkgDiscussion) GetCurrentStage() Stage {
	if d.Deal == nil {
		return StageStart
	}
	return StageDealReceived

}
