package model

type ConsensusReachedListener interface {
	GetConsensusDecisionMadeEventChannel() chan ConsensusDecision
}
