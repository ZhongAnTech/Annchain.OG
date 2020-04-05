package hotstuff_event

import "github.com/sirupsen/logrus"

type Safety struct {
	lastVoteRound  int
	preferredRound int
	Ledger         *Ledger
	Partner        *Partner
	Logger         *logrus.Logger
	//	voteMsg := &Msg{
	//	Typev:    Vote,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderId: nil,
	//	Id:       msg.Id,
	//}
}

func (s *Safety) UpdatePreferredRound(qc *QC) {
	if qc.VoteData.ParentRound > s.preferredRound {
		s.Logger.WithField("qc", qc).Trace("update preferred round")
		s.preferredRound = qc.VoteData.ParentRound
	}
}

func (s *Safety) MakeVote(blockId string, blockRound int, parentQC *QC) *ContentVote {
	// This function exercises both the voting and the commit rules
	if blockRound < s.lastVoteRound || parentQC.VoteData.Round < s.preferredRound {
		return nil
	}
	s.IncreaseLastVoteRound(blockRound)
	s.Partner.SaveConsensusState()

	// VoteINfo carries the potential QC info with ids and rounds of the whole three-chain
	voteInfo := VoteInfo{
		Id:               blockId,
		Round:            blockRound,
		ParentId:         parentQC.VoteData.Id,
		ParentRound:      parentQC.VoteData.Round,
		GrandParentId:    parentQC.VoteData.ParentId,
		GrandParentRound: parentQC.VoteData.ParentRound,
		ExecStateId:      s.Ledger.GetState(blockId),
	}
	potentialCommitId := s.CommitRule(parentQC, blockRound) // TODO: might be empty string

	ledgerCommitInfo := LedgerCommitInfo{
		CommitStateId: s.Ledger.GetState(potentialCommitId),
		VoteInfoHash:  Hash(voteInfo.GetHashContent()),
	}

	return &ContentVote{
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

func (s *Safety) CommitRule(qc *QC, voteRound int) string {
	// find the committed id in case a qc is formed in the vote round
	if qc.VoteData.ParentRound+1 == qc.VoteData.Round && qc.VoteData.Round+1 == voteRound {
		return qc.VoteData.ParentId
	} else {
		return ""
	}
}
