package hotstuff_event

type Safety struct {
	lastVoteRound  int
	preferredRound int
	Ledger         *Ledger
	//	voteMsg := &Msg{
	//	Typev:    VOTE,
	//	ParentQC: msg.ParentQC,
	//	Round:    msg.Round,
	//	SenderId: nil,
	//	Id:       msg.Id,
	//}
}

func (s *Safety) UpdatePreferredRound(qc *QC) {
	if qc.VoteInfo.ParentRound > s.preferredRound {
		s.preferredRound = qc.VoteInfo.ParentRound
	}
}

func (s *Safety) MakeVote(blockId string, blockRound int, parentQC *QC) *ContentVote {
	// This function exercises both the voting and the commit rules
	if blockRound < s.lastVoteRound || parentQC.VoteInfo.Round < s.preferredRound {
		return nil
	}
	s.IncreaseLastVoteRound(blockRound)
	SaveConsensusState()

	voteInfo := VoteInfo{
		Id:          blockId,
		Round:       blockRound,
		ParentId:    parentQC.VoteInfo.Id,
		ParentRound: parentQC.VoteInfo.Round,
		ExecStateId: s.Ledger.GetState(blockId),
	}
	potentialCommitId := s.CommitRule(parentQC, blockRound) // TODO: might be empty string

	ledgerCommitInfo := LedgerCommitInfo{
		CommitStateId: s.Ledger.GetState(potentialCommitId),
		VoteInfoHash:  Hash(voteInfo.GetHashContent()),
	}

	return &ContentVote{
		VoteInfo:         voteInfo,
		LedgerCommitInfo: ledgerCommitInfo,
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
	if qc.VoteInfo.ParentRound+1 == qc.VoteInfo.Round && qc.VoteInfo.Round+1 == voteRound {
		return qc.VoteInfo.ParentId
	} else {
		return ""
	}
}
