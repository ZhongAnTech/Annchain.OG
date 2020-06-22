package consensus

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/sirupsen/logrus"
)

type Safety struct {
	lastVoteRound  int64
	preferredRound int64
	Ledger         *Ledger
	Partner        *Partner
	Logger         *logrus.Logger
	//	voteMsg := &Msg{
	//	Typev:    HotStuffMessageTypeVote,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderId: nil,
	//	Id:       msg.Id,
	//}
}

func (s *Safety) UpdatePreferredRound(qc *consensus_interface.QC) {
	if qc.VoteData.ParentRound > s.preferredRound {
		s.Logger.WithField("qc", qc).Trace("update preferred round")
		s.preferredRound = qc.VoteData.ParentRound
		s.Partner.Report.Report("PreferredRound", s.preferredRound, false)
	}
}

func (s *Safety) MakeVote(blockId string, blockRound int, parentQC *consensus_interface.QC) *consensus_interface.ContentVote {
	// This function exercises both the voting and the commit rules
	if blockRound < s.lastVoteRound || parentQC.VoteData.Round < s.preferredRound {
		return nil
	}
	s.IncreaseLastVoteRound(blockRound)
	s.Partner.SaveConsensusState()

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
	potentialCommitId := s.CommitRule(parentQC, blockRound) // TODO: might be empty string

	ledgerCommitInfo := consensus_interface.LedgerCommitInfo{
		CommitStateId: s.Ledger.GetState(potentialCommitId),
		VoteInfoHash:  Hash(voteInfo.GetHashContent()),
	}

	return &consensus_interface.ContentVote{
		VoteInfo:         voteInfo,
		LedgerCommitInfo: ledgerCommitInfo,
		QC:               nil,
		TC:               nil,
	}
}

// IncreaseLastVoteRound
func (s *Safety) IncreaseLastVoteRound(targetRound int) {
	// commit not to vote in rounds lower than target
	if s.lastVoteRound < targetRound {
		s.lastVoteRound = targetRound
	}
}

func (s *Safety) CommitRule(qc *consensus_interface.QC, voteRound int) string {
	// find the committed id in case a qc is formed in the vote round
	if qc.VoteData.ParentRound+1 == qc.VoteData.Round && qc.VoteData.Round+1 == voteRound {
		return qc.VoteData.ParentId
	} else {
		return ""
	}
}
