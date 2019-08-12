package model

type ConsensusReachedListener interface {
	GetEventChannel() chan Proposal
}
