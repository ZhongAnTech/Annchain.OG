package hotstuff_event

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
)

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

type Hashable interface {
	GetHashContent() string
}

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

type Content interface {
	fmt.Stringer
	SignatureTarget() string
}

type ContentProposal struct {
	Proposal Block
}

func (c ContentProposal) String() string {
	return fmt.Sprintf("[CProposal: %s]", c.Proposal)
}

func (c ContentProposal) SignatureTarget() string {
	return c.Proposal.String()
}

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

type ContentLocalTimeout struct {
}

func (c ContentLocalTimeout) String() string {
	return ""
}

func (c ContentLocalTimeout) SignatureTarget() string {
	panic("implement me")
}

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

type Signature struct {
	PartnerId int
	Signature string
}

func (s Signature) String() string {
	return fmt.Sprintf("[Sig: partnerId %d]", s.PartnerId)
}

func Hash(s string) string {
	bs := md5.Sum([]byte(s))
	return hex.EncodeToString(bs[:])
}

type NilInt struct {
	Value int
}
