package consensus

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
)

type Safety struct {
	Ledger   consensus_interface.Ledger
	Reporter *soccerdash.Reporter
	Logger   *logrus.Logger
	Hasher   consensus_interface.Hasher

	consensusState *consensus_interface.ConsensusState

	//	voteMsg := &Msg{
	//	Typev:    HotStuffMessageTypeVote,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderMemberId: nil,
	//	Id:       msg.Id,
	//}
}

func (s *Safety) ConsensusState() *consensus_interface.ConsensusState {
	return s.consensusState
}

func (s *Safety) SetConsensusState(consensusState *consensus_interface.ConsensusState) {
	s.consensusState = consensusState
}

func (s *Safety) InitDefault() {
	s.consensusState = s.Ledger.GetConsensusState()
}

func (s *Safety) UpdatePreferredRound(qc *consensus_interface.QC) {
	if qc.VoteData.ParentRound > s.consensusState.PreferredRound {
		s.Logger.WithField("qc", qc).Trace("update preferred round")
		s.consensusState.PreferredRound = qc.VoteData.ParentRound
		s.Reporter.Report("PreferredRound", s.consensusState.PreferredRound, false)
	}
}

func (s *Safety) MakeVote(blockId string, blockRound int64, parentQC *consensus_interface.QC) *consensus_interface.ContentVote {
	// This function exercises both the voting and the commit rules
	if blockRound < s.consensusState.LastVoteRound || parentQC.VoteData.Round < s.consensusState.PreferredRound {
		return nil
	}
	s.IncreaseLastVoteRound(blockRound)
	s.Ledger.SaveConsensusState(s.consensusState)

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

// IncreaseLastVoteRound
func (s *Safety) IncreaseLastVoteRound(targetRound int64) {
	// commit not to vote in rounds lower than target
	if s.consensusState.LastVoteRound < targetRound {
		s.consensusState.LastVoteRound = targetRound
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

func (s *Safety) SetHighQC(qc *consensus_interface.QC) {
	s.consensusState.HighQC = qc
	s.Ledger.SaveConsensusState(s.consensusState)
}
