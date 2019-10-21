// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ogmessage

import (
	"fmt"
	"github.com/annchain/OG/common"
	"strings"
)

//go:generate msgp

//msgp:tuple SequencerHeader
type SequencerHeader struct {
	Hash   common.Hash
	Height uint64
}

//msgp:tuple SequencerHeaders
type SequencerHeaders []*SequencerHeader

func (s *SequencerHeader) SequencerId() uint64 {
	return s.Height
}

func (s *SequencerHeader) GetHash() common.Hash {
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

func NewSequencerHead(hash common.Hash, height uint64) *SequencerHeader {
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
