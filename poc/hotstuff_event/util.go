package hotstuff_event

import (
	"crypto/md5"
	"encoding/hex"
	"strconv"
)

type MsgType int

const (
	PROPOSAL MsgType = iota
	VOTE
	TIMEOUT
	LOCAL_TIMEOUT
)

type Hashable interface {
	GetHashContent() string
}

// Node is a block
type Node struct {
	Previous string
	content  string
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

type Content interface {
	SignatureTarget() string
}

type ContentProposal struct {
	Proposal Block
}

func (c ContentProposal) SignatureTarget() string {
	panic("implement me")
}

type ContentTimeout struct {
	Round int
}

func (c ContentTimeout) SignatureTarget() string {
	return strconv.Itoa(c.Round)
}

type ContentVote struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
}

func (c ContentVote) SignatureTarget() string {
	panic("implement me")
}

type ContentLocalTimeout struct {
}

func (c ContentLocalTimeout) SignatureTarget() string {
	panic("implement me")
}

type QC struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	GrandParentId    int
	Signatures       []Signature // simulate sig by give an id to the vote
	//Typev      MsgType
	//ViewNumber int
	//Node       *Node

}

type Signature struct {
	PartnerId int
	Signature string
}

func Hash(s string) string {
	bs := md5.Sum([]byte(s))
	return hex.EncodeToString(bs[:])
}

type NilInt struct {
	Value int
}
