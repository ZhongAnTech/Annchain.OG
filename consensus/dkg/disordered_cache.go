package dkg

import (
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
type DisorderedCache map[uint32]Stagable

// Stagable will justify which stage it is currently on, depending on the messages it received
// e.g., if the messages received are 1,2,6,7,8, then the current stage is 2
type Stagable interface {
	GetCurrentStage() Stage
}

type DkgDiscussion struct {
	Deal      *dkg.Deal
	Responses []*dkg.Response
}

func (d *DkgDiscussion) GetCurrentStage() Stage {
	if d.Deal == nil {
		return StageStart
	}
	return StageDealReceived
}
