package model

import (
	"crypto/sha256"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/types/msg"
	"github.com/annchain/OG/types/tx_types"
)

//go:generate msgp

type Proposal interface {
	Equal(Proposal) bool
	GetId() *common.Hash
	msg.Message
	String() string
	Copy() Proposal
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

func (s StringProposal) Copy() Proposal {
	var r StringProposal
	r = s
	return &r
}

func (s StringProposal) GetId() *common.Hash {
	h := sha256.New()
	h.Write([]byte(s))
	sum := h.Sum(nil)
	hash := common.Hash{}
	hash.MustSetBytes(sum, common.PaddingNone)
	return &hash
}

func (s StringProposal) String() string {
	return string(s)
}

//msgp:tuple SequencerProposal
type SequencerProposal struct {
	tx_types.Sequencer
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

func (s SequencerProposal) GetId() *common.Hash {
	//should copy ?
	var hash common.Hash
	hash.MustSetBytes(s.GetTxHash().ToBytes(), common.PaddingNone)
	return &hash
}

func (s SequencerProposal) Copy() Proposal {
	var r SequencerProposal
	r = s
	return &r
}
