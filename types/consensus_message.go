package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/common/msg"
)

//go:generate msgp

// HeightRound is the current progress of the consensus.
// Height is the block height, round is the sub-progress if no consensus can be easily reached
//msgp:tuple HeightRound
type HeightRound struct {
	Height uint64
	Round  int
}

func (h *HeightRound) String() string {
	return fmt.Sprintf("[%d-%d]", h.Height, h.Round)
}

// IsAfter judges whether self is a higher HeightRound
func (h *HeightRound) IsAfter(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round > o.Round)
}

// IsAfterOrEqual judges whether self is a higher or equal HeightRound
func (h *HeightRound) IsAfterOrEqual(o HeightRound) bool {
	return h.Height > o.Height ||
		(h.Height == o.Height && h.Round >= o.Round)
}

// IsAfterOrEqual judges whether self is a lower HeightRound
func (h *HeightRound) IsBefore(o HeightRound) bool {
	return h.Height < o.Height ||
		(h.Height == o.Height && h.Round < o.Round)
}

type Proposal interface {
	Equal(Proposal) bool
	GetId() *Hash
	msg.Message
	String() string
	Copy()Proposal
}

//StringProposal is for test
type StringProposal string

func (s StringProposal) Equal(o Proposal) bool {
	v, ok := o.(*StringProposal)
	if !ok || v == nil {
		return false
	}
	return s == *v
}

func (s StringProposal)Copy ()Proposal {
	var r StringProposal
	r = s
	return &r
}

func (s StringProposal) GetId() *Hash {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	hash := Hash{}
	hash.MustSetBytes(sum, PaddingNone)
	return &hash
}

func (s StringProposal) String() string {
	return string(s)
}

type SequencerProposal struct {
	Sequencer
}

//msgp:tuple BasicMessage
type BasicMessage struct {
	SourceId    uint16
	HeightRound HeightRound
}

//msgp:tuple MessageProposal
type MessageProposal struct {
	BasicMessage
	Value      Proposal //TODO
	ValidRound int
	//PublicKey  []byte
	Signature []byte
}

func (m  MessageProposal)Copy()*MessageProposal {
	var r MessageProposal
	r = m
	r.Value = m.Value.Copy()
	return  &r
}

//msgp:tuple MessageCommonVote
type MessagePreVote struct {
	BasicMessage
	Idv       *Hash // ID of the proposal, usually be the hash of the proposal
	Signature []byte
	PublicKey []byte
}

type MessagePreCommit struct {
	BasicMessage
	Idv          *Hash // ID of the proposal, usually be the hash of the proposal
	BlsSignature []byte
	Signature    []byte
	PublicKey    []byte
}

func (m BasicMessage) String() string {
	return fmt.Sprintf("SourceId:%d ,hr:%d", m.SourceId, m.HeightRound)
}

func (m MessageProposal) String() string {
	return fmt.Sprintf("bm %s, value %s, vaildRound %d", m.BasicMessage, m.Value, m.ValidRound)
}

func (m MessagePreVote) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BasicMessage, m.Idv)
}

func (m MessagePreCommit) String() string {
	return fmt.Sprintf("bm %s, idv %s", m.BasicMessage, m.Idv)
}

func (m *MessagePreVote) SignatureTargets() []byte {
	var buf bytes.Buffer
	if m.Idv != nil {
		panicIfError(binary.Write(&buf, binary.BigEndian, m.Idv.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, m.HeightRound.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, uint64(m.HeightRound.Round)))
	panicIfError(binary.Write(&buf, binary.BigEndian, m.SourceId))
	return buf.Bytes()
}

func (m *MessagePreCommit) SignatureTargets() []byte {
	var buf bytes.Buffer
	panicIfError(binary.Write(&buf, binary.BigEndian, m.BlsSignature))
	return buf.Bytes()
}

func (m *MessagePreCommit) BlsSignatureTargets() []byte {
	var buf bytes.Buffer
	if m.Idv != nil {
		panicIfError(binary.Write(&buf, binary.BigEndian, m.Idv.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, m.HeightRound.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, uint64(m.HeightRound.Round)))
	panicIfError(binary.Write(&buf, binary.BigEndian, m.SourceId))
	return buf.Bytes()
}

func (m *MessageProposal) SignatureTargets() []byte {
	var buf bytes.Buffer
	if idv := m.Value.GetId(); idv != nil {
		panicIfError(binary.Write(&buf, binary.BigEndian, idv.Bytes))
	}
	panicIfError(binary.Write(&buf, binary.BigEndian, m.HeightRound.Height))
	panicIfError(binary.Write(&buf, binary.BigEndian, uint64(m.HeightRound.Round)))
	panicIfError(binary.Write(&buf, binary.BigEndian, uint64(m.ValidRound)))
	panicIfError(binary.Write(&buf, binary.BigEndian, m.SourceId))
	return buf.Bytes()
}

func (s *SequencerProposal) String() string {
	return fmt.Sprintf("seqProposal") + s.Sequencer.String()
}

func (s SequencerProposal) Equal(o Proposal) bool {
	v, ok := o.(*SequencerProposal)
	if !ok || v == nil {
		return false
	}
	return s.GetTxHash() == v.GetTxHash()
}

func (s SequencerProposal) GetId() *Hash {
	//should copy ?
	var hash Hash
	hash.MustSetBytes(s.GetTxHash().ToBytes(), PaddingNone)
	return &hash
}


func (s SequencerProposal)Copy()Proposal {
	var r SequencerProposal
	r =s
	return &r
}