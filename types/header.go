package types

import (
	"fmt"
	"strings"
)

//go:generate msgp

type SequencerHeader struct {
	hash Hash
	id   uint64
}

type SequencerHeaders []*SequencerHeader

func (s *SequencerHeader) SequencerId() uint64 {
	return s.id
}

func (s *SequencerHeader) Hash() Hash {
	return s.hash
}

func (s *SequencerHeader) Id() uint64 {
	return s.id
}

func (s *SequencerHeader) String() string {
	if s == nil {
		return fmt.Sprintf("nil")
	}
	return fmt.Sprintf("%d-[%.10s]", s.Id(), s.Hash().Hex())
}

func (s *SequencerHeader) StringFull() string {
	if s == nil {
		return fmt.Sprintf("nil")
	}
	return fmt.Sprintf("%d-[%s]", s.Id(), s.Hash().Hex())
}

func NewSequencerHead(hash Hash, id uint64) *SequencerHeader {
	return &SequencerHeader{
		hash: hash,
		id:   id,
	}
}


func (s *SequencerHeader) Equal(h *SequencerHeader) bool {
	if s == nil || h == nil {
		return false
	}
	return s.id == h.id && s.hash == h.hash
}


func (h SequencerHeaders)String()string{
	var strs []string
	for _, v := range h {
		strs = append(strs,v.String())
	}
	return strings.Join(strs, ", ")
}