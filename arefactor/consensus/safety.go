package consensus

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"sync"
)

type Safety struct {
	Ledger   consensus_interface.Ledger
	Reporter *soccerdash.Reporter
	Logger   *logrus.Logger
	Hasher   consensus_interface.Hasher

	consensusState   *consensus_interface.ConsensusState // Only safety can manage consensusState
	consensusStateMu sync.RWMutex

	//	voteMsg := &Msg{
	//	Typev:    HotStuffMessageTypeVote,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderMemberId: nil,
	//	Id:       msg.Id,
	//}
}

func (s *Safety) ConsensusState() *consensus_interface.ConsensusState {
	s.consensusStateMu.RLock()
	defer s.consensusStateMu.RUnlock()
	return s.consensusState
}

func (s *Safety) InitDefault() {
	s.consensusState = s.Ledger.GetConsensusState()
}

func (s *Safety) UpdatePreferredRound(qc *consensus_interface.QC) {
	if qc.VoteData.ParentRound > s.consensusState.PreferredRound {
		s.SetPreferredRound(qc.VoteData.ParentRound)
	}
}

func (s *Safety) MakeVote(blockId string, blockRound int64, parentQC *consensus_interface.QC) *consensus_interface.ContentVote {
	// This function exercises both the voting and the commit rules
	if blockRound < s.consensusState.LastVoteRound || parentQC.VoteData.Round < s.consensusState.PreferredRound {
		logrus.WithFields(logrus.Fields{
			"blockId":        blockId,
			"blockRound":     blockRound,
			"parentQc":       parentQC,
			"consensusState": s.consensusState,
		}).Warn("I don't vote this proposal.")
		return nil
	}
	s.IncreaseLastVoteRound(blockRound)

	// VoteINfo carries the potential QC info with ids and rounds of the whole three-chain
	voteInfo := consensus_interface.VoteInfo{
		Id:               blockId,
		Round:            blockRound,
		ParentId:         parentQC.VoteData.Id,
		ParentRound:      parentQC.VoteData.Round,
		GrandParentId:    parentQC.VoteData.ParentId,
		GrandParentRound: parentQC.VoteData.ParentRound,
		ExecStateId:      s.Ledger.GetState(blockId),
	}
	// TODO: check if execStateId is ""

	potentialCommitId := s.CommitRule(parentQC, blockRound) // TODO: might be empty string

	ledgerCommitInfo := consensus_interface.LedgerCommitInfo{
		CommitStateId: s.Ledger.GetState(potentialCommitId),
		VoteInfoHash:  s.Hasher.Hash(voteInfo.GetHashContent()),
	}
	// check if CommitStateId is ""

	return &consensus_interface.ContentVote{
		VoteInfo:         voteInfo,
		LedgerCommitInfo: ledgerCommitInfo,
		QC:               nil,
		TC:               nil,
	}
}

func (s *Safety) CommitRule(qc *consensus_interface.QC, voteRound int64) string {
	// find the committed id in case a qc is formed in the vote round
	if qc.VoteData.ParentRound+1 == qc.VoteData.Round && qc.VoteData.Round+1 == voteRound {
		return qc.VoteData.ParentId
	} else {
		return ""
	}
}

// IncreaseLastVoteRound
func (s *Safety) IncreaseLastVoteRound(targetRound int64) {
	// commit not to vote in rounds lower than target
	if s.consensusState.LastVoteRound < targetRound {
		s.SetLastVoteRound(targetRound)
	}
}

func (s *Safety) SetLastVoteRound(round int64) {
	s.consensusStateMu.Lock()
	defer s.consensusStateMu.Unlock()
	logrus.WithField("from", s.consensusState.LastVoteRound).WithField("to", round).Trace("update lastVoteRound")
	s.consensusState.LastVoteRound = round
	s.Ledger.SaveConsensusState(s.consensusState)

	s.Reporter.Report("LastVoteRound", s.consensusState.LastVoteRound, false)
}

func (s *Safety) SetHighQC(qc *consensus_interface.QC) {
	s.consensusStateMu.Lock()
	defer s.consensusStateMu.Unlock()
	logrus.WithField("from", s.consensusState.HighQC).WithField("to", qc).Trace("update highQC")
	s.consensusState.HighQC = qc
	s.Ledger.SaveConsensusState(s.consensusState)

	s.Reporter.Report("HighQC", s.consensusState.HighQC, false)
}

func (s *Safety) SetLastTC(tc *consensus_interface.TC) {
	s.consensusStateMu.Lock()
	defer s.consensusStateMu.Unlock()
	logrus.WithField("from", s.consensusState.LastTC).WithField("to", tc).Trace("update lastTC")
	s.consensusState.LastTC = tc
	s.Ledger.SaveConsensusState(s.consensusState)

	s.Reporter.Report("LastTC", s.consensusState.LastTC, false)
}

func (s *Safety) SetPreferredRound(round int64) {
	s.consensusStateMu.Lock()
	defer s.consensusStateMu.Unlock()
	logrus.WithField("from", s.consensusState.PreferredRound).WithField("to", round).Trace("update lastTC")
	s.consensusState.PreferredRound = round
	s.Ledger.SaveConsensusState(s.consensusState)

	s.Reporter.Report("PreferredRound", s.consensusState.PreferredRound, false)
}
