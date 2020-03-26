package hotstuff_event

import (
	"fmt"
	"strconv"
)

//go:generate msgp

type MsgType int

func (m MsgType) String() string {
	switch m {
	case Proposal:
		return "Proposal"
	case Vote:
		return "Vote"
	case Timeout:
		return "Timeout"
	case LocalTimeout:
		return "LocalTimeout"
	default:
		return "NAMsg"
	}
}

const (
	Proposal MsgType = iota
	Vote
	Timeout
	LocalTimeout
)

//msgp:tuple Msg
type Msg struct {
	Typev    MsgType
	Sig      Signature
	SenderId int
	Content  Content
	//ParentQC *QC
	//Round    int
	//Id       int //

	//ViewNumber    int
	//Node          *Node
	//Justify       *QC
	//FromPartnerId int
}

func (m Msg) String() string {
	return fmt.Sprintf("[type:%s sender=%d content=%s sig=%s]", m.Typev, m.SenderId, m.Content, m.Sig)
}

//msgp:tuple Block
type Block struct {
	Round    int    // the round that generated this proposal
	Payload  string // proposed transactions
	ParentQC *QC    // qc for parent block
	Id       string // unique digest of round, payload and parent_qc.id
}

func (b Block) String() string {
	return fmt.Sprintf("[Block round=%d payload=%s parentQC=%s id=%s]", b.Round, b.Payload, b.ParentQC, b.Id)
}

//msgp:tuple ContentProposal
type ContentProposal struct {
	Proposal Block
}

func (c ContentProposal) String() string {
	return fmt.Sprintf("[CProposal: %s]", c.Proposal)
}

func (c ContentProposal) SignatureTarget() string {
	return c.Proposal.String()
}

//msgp:tuple ContentTimeout
type ContentTimeout struct {
	Round  int
	HighQC *QC
}

func (c ContentTimeout) SignatureTarget() string {
	return strconv.Itoa(c.Round)
}

func (c ContentTimeout) String() string {
	return fmt.Sprintf("[CTimeout: round=%d highQC=%s]", c.Round, c.HighQC)
}

//msgp:tuple ContentVote
type ContentVote struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	Signatures       []Signature
}

func (c ContentVote) String() string {
	return fmt.Sprintf("[CVote: voteInfo=%s ledgerCommitInfo=%s]", c.VoteInfo, c.LedgerCommitInfo)
}

func (c ContentVote) SignatureTarget() string {
	return fmt.Sprintf("[CVote: voteInfo=%s ledgerCommitInfo=%s]", c.VoteInfo, c.LedgerCommitInfo)
}

//type ContentLocalTimeout struct {
//}
//
//func (c ContentLocalTimeout) String() string {
//	return ""
//}
//
//func (c ContentLocalTimeout) SignatureTarget() string {
//	panic("implement me")
//}

//msgp:tuple QC
type QC struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	Signatures       []Signature // simulate sig by give an id to the vote
	//Typev      MsgType
	//ViewNumber int
	//Node       *Node

}

func (q QC) String() string {
	return fmt.Sprintf("[QC: voteInfo=%s ledgerCommitInfo=%s sigs=%d]", q.VoteInfo, q.LedgerCommitInfo, len(q.Signatures))
}

//msgp:tuple Signature
type Signature struct {
	PartnerId int
	Signature string
}

func (s Signature) String() string {
	return fmt.Sprintf("[Sig: partnerId %d]", s.PartnerId)
}

//msgp:tuple VoteInfo
type VoteInfo struct {
	Id               string // Id of the block
	Round            int    // round of the block
	ParentId         string // Id of the parent
	ParentRound      int    // round of the parent
	GrandParentId    string // Id of the grandParent
	GrandParentRound int    // round of the grandParent
	ExecStateId      string // speculated execution state
}

func (i VoteInfo) String() string {
	return fmt.Sprintf("<%s %d> <%s %d> <%s %d> %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

func (i VoteInfo) GetHashContent() string {
	return fmt.Sprintf("%s %d %s %d %s %d %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

//msgp:tuple LedgerCommitInfo
type LedgerCommitInfo struct {
	CommitStateId string // nil if no commit happens when this vote is aggregated to QC. Usually the merkle root
	VoteInfoHash  string // hash of VoteMsg.voteInfo
}

func (l LedgerCommitInfo) String() string {
	return fmt.Sprintf("[LedgerCommitInfo: commitStateId %s VoteInfoHash %s]", l.CommitStateId, l.VoteInfoHash)
}

func (l LedgerCommitInfo) GetHashContent() string {
	return fmt.Sprintf("%s %s", l.CommitStateId, l.VoteInfoHash)
}

//msgp:tuple VoteMsg
type VoteMsg struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	Sender           int
	Signature        Signature
}
