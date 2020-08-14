package consensus_interface

import (
	"fmt"
	"github.com/annchain/commongo/hexutil"
	"strconv"
)

//go:generate msgp

type HotStuffMessageType int

const HotStuffMessageTypeRoot = 100

const (
	HotStuffMessageTypeProposal HotStuffMessageType = iota + 100
	HotStuffMessageTypeVote
	HotStuffMessageTypeTimeout
	HotStuffMessageTypeString
)

func (m HotStuffMessageType) String() string {
	switch m {
	case HotStuffMessageTypeProposal:
		return "HotStuffProposal"
	case HotStuffMessageTypeVote:
		return "HotStuffVote"
	case HotStuffMessageTypeTimeout:
		return "HotStuffTimeout"
	//case LocalTimeout:
	//	return "LocalTimeout"
	case HotStuffMessageTypeString:
		return "HotStuffString"
	default:
		return "Unknown Message " + strconv.Itoa(int(m))
	}
}

//msgp Block
type Block struct {
	Round    int64  // the round that generated this proposal
	Payload  string // proposed transactions
	ParentQC *QC    // qc for parent block
	Id       string // unique digest of round, payload and parent_qc.id
}

func (b Block) String() string {
	return fmt.Sprintf("[Block round=%d payload=%s parentQC=%s id=%s]", b.Round, b.Payload, b.ParentQC, b.Id)
}

// HotStuffSignedMessage is for transportation.
//msgp HotStuffSignedMessage
type HotStuffSignedMessage struct {
	HotStuffMessageType int    // what ContentByte is (one of HotStuffMessageType).
	ContentBytes        []byte // this Byte will be recovered to implementation of Content interface
	SenderMemberId      string // member id of the sender
	Signature           []byte
}

func (z *HotStuffSignedMessage) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *HotStuffSignedMessage) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *HotStuffSignedMessage) GetTypeValue() int {
	return HotStuffMessageTypeRoot
}

func (z *HotStuffSignedMessage) String() string {
	return fmt.Sprintf("WM: Type=%s SenderMemberId=%s ContentBytes=%s",
		HotStuffMessageType(z.HotStuffMessageType).String(),
		z.SenderMemberId,
		hexutil.ToBriefHex(z.ContentBytes, 100))
}

//msgp Signature
type Signature []byte

//msgp JointSignature
type JointSignature []byte

//msgp ContentString
// it is a dummy content type for debugging
type ContentString struct {
	Content string
}

func (z *ContentString) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *ContentString) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *ContentString) String() string {
	return z.Content
}

func (z *ContentString) SignatureTarget() []byte {
	return z.ToBytes()
}

//msgp ContentProposal
type ContentProposal struct {
	Proposal Block
	TC       *TC
}

func (z *ContentProposal) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *ContentProposal) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *ContentProposal) String() string {
	return fmt.Sprintf("[CProposal: p=%s tc=%s]", z.Proposal, z.TC)
}

func (z *ContentProposal) SignatureTarget() []byte {
	return z.ToBytes()
}

//msgp ContentTimeout
type ContentTimeout struct {
	Round  int64
	HighQC *QC
	TC     *TC
}

func (z *ContentTimeout) SignatureTarget() []byte {
	return z.ToBytes()
}

func (z *ContentTimeout) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *ContentTimeout) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *ContentTimeout) String() string {
	return fmt.Sprintf("[CTimeout: round=%d highQC=%s tc=%s]", z.Round, z.HighQC, z.TC)
}

//msgp ContentVote
type ContentVote struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	QC               *QC
	TC               *TC
}

func (z *ContentVote) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *ContentVote) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *ContentVote) String() string {
	return fmt.Sprintf("[CVote: voteinfo=%s ledgercinfo=%s qc=%s tc=%s]", z.VoteInfo, z.LedgerCommitInfo, z.QC, z.TC)
}

func (z *ContentVote) SignatureTarget() []byte {
	return z.ToBytes()
}

//msgp QC
type QC struct {
	VoteData       VoteInfo
	JointSignature JointSignature
}

func (q *QC) String() string {
	if q == nil {
		return "nil"
	}
	return fmt.Sprintf("[QC: voteInfo=%s sigs=%d]", q.VoteData, len(q.JointSignature))
}

//msgp TC
type TC struct {
	Round          int64 // round of the block
	JointSignature JointSignature
}

func (t *TC) String() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("[TC: round=%d, sigs=%d]", t.Round, len(t.JointSignature))
}

//msgp VoteInfo
type VoteInfo struct {
	Id               string // Id of the block
	Round            int64  // round of the block
	ParentId         string // Id of the parent
	ParentRound      int64  // round of the parent
	GrandParentId    string // Id of the grandParent
	GrandParentRound int64  // round of the grandParent
	ExecStateId      string // speculated execution state
}

func (i VoteInfo) String() string {
	return fmt.Sprintf("<%s %d> <%s %d> <%s %d> %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

func (i VoteInfo) GetHashContent() string {
	return fmt.Sprintf("%s %d %s %d %s %d %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

//msgp LedgerCommitInfo
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
