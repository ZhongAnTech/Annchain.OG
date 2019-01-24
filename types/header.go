package types

import (
	"fmt"
	"strings"
)

//go:generate msgp

type SequencerHeader struct {
	Hash   Hash
	Height uint64
}

type SequencerHeaders []*SequencerHeader

func (s *SequencerHeader) SequencerId() uint64 {
	return s.Height
}

func (s *SequencerHeader) GetHash() Hash {
	return s.Hash
}

func (s *SequencerHeader) GetHeight() uint64 {
	return s.Height
}

func (s *SequencerHeader) String() string {
	if s == nil {
		return fmt.Sprintf("nil")
	}
	return fmt.Sprintf("%d-[%.10s]", s.Height, s.GetHash().Hex())
}

func (s *SequencerHeader) StringFull() string {
	if s == nil {
		return fmt.Sprintf("nil")
	}
	return fmt.Sprintf("%d-[%s]", s.GetHeight(), s.GetHash().Hex())
}

func NewSequencerHead(hash Hash, height uint64) *SequencerHeader {
	return &SequencerHeader{
		Hash:   hash,
		Height: height,
	}
}

func (s *SequencerHeader) Equal(h *SequencerHeader) bool {
	if s == nil || h == nil {
		return false
	}
	return s.Height == h.Height && s.Hash == h.Hash
}

func (h SequencerHeaders) String() string {
	var strs []string
	for _, v := range h {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}
