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
package annsensus

import (
	"sync"

	"github.com/annchain/OG/types"
)

type Term struct {
	id             uint64   `json:"id"`
	flag           bool      
	partsNum       int
	senators       Senators  `json:"senators"`
	formerSenators map[uint64]Senators `json:"former_senators"`
	candidates     map[types.Address]*types.Campaign
	alsorans       map[types.Address]*types.Campaign

	mu sync.RWMutex
}

func newTerm(id uint64, pn int) *Term {
	return &Term{
		id:             id,
		flag:           false,
		partsNum:       pn,
		senators:       make(Senators),
		formerSenators: make(map[uint64]Senators),
		candidates:     make(map[types.Address]*types.Campaign),
		alsorans:       make(map[types.Address]*types.Campaign),
	}
}

func (t *Term) ID() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.id
}

func (t *Term) UpdateID(id uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.id = id
}

func (t *Term) SwitchFlag(flag bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.flag = flag
}

func (t *Term) Changing() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.flag
}

func (t *Term) GetCandidate(addr types.Address) *types.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.candidates[addr]
}

func (t *Term) Candidates() map[types.Address]*types.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.candidates
}

func (t *Term) AddCandidate(c *types.Campaign) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.candidates[c.Issuer] = c
}

func (t *Term) GetAlsoran(addr types.Address) *types.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.alsorans[addr]
}

func (t *Term) Alsorans() map[types.Address]*types.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.alsorans
}

func (t *Term) AddAlsorans(camps []*types.Campaign) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, c := range camps {
		if c == nil {
			continue
		}
		// TODO
		// this check is not proper enough, try optimize it.
		if t.hasCampaign(c) {
			continue
		}
		t.alsorans[c.Issuer] = c
	}
}

func (t *Term) HasCampaign(c *types.Campaign) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.hasCampaign(c)
}

func (t *Term) hasCampaign(c *types.Campaign) bool {
	if _, exists := t.candidates[c.Issuer]; exists {
		return true
	}
	if _, exists := t.alsorans[c.Issuer]; exists {
		return true
	}
	return false
}

// CanChange returns true if the campaigns cached reaches the
// term change requirments.
func (t *Term) CanChange() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO:
	// term change requirements are not enough now.
	if len(t.candidates) == 0 {
		return false
	}
	if len(t.candidates) < t.partsNum {
		log.WithField("len ", len(t.candidates)).Debug("not enough campaigns , waiting")
		return false
	}

	return true
}

func (t *Term) ChangeTerm(tc *types.TermChange) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	snts := make(map[types.Address]*Senator)
	for addr, c := range t.candidates {
		s := newSenator(addr, c.PublicKey, tc.PkBls)
		snts[addr] = s
	}

	t.candidates = make(map[types.Address]*types.Campaign)
	t.alsorans = make(map[types.Address]*types.Campaign)

	formerSnts := t.senators
	t.formerSenators[t.id] = formerSnts

	t.senators = snts

	// TODO
	// 1. update id.
	// 2. process alsorans.

	return nil
}

type Senator struct {
	addr  types.Address
	pk    []byte
	blspk []byte
	// TODO:
	// more variables?
}

func newSenator(addr types.Address, publickey, blspk []byte) *Senator {
	s := &Senator{}
	s.addr = addr
	s.pk = publickey
	s.blspk = blspk

	return s
}

type Senators map[types.Address]*Senator

func (t *Term) GetSenater(address types.Address) *Senator {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if v, ok := t.senators[address]; ok {
		return v
	}
	return nil
}
