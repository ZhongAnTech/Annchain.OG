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
package term

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/campaign"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Term struct {
	id                     uint32 `json:"id"`
	flag                   bool
	partsNum               int
	senators               Senators            `json:"senators"`
	formerSenators         map[uint32]Senators `json:"former_senators"`
	candidates             map[common.Address]*campaign.Campaign
	publicKeys             []crypto.PublicKey
	formerPublicKeys       []crypto.PublicKey
	alsorans               map[common.Address]*campaign.Campaign
	campaigns              map[common.Address]*campaign.Campaign
	startedHeight          uint64
	generateCampaignHeight uint64
	newTerm                bool
	termChangeInterval     int

	mu                sync.RWMutex
	currentTermChange *campaign.TermChange
	genesisTermChange *campaign.TermChange
	started           bool
}

func NewTerm(id uint32, participantNumber int, termChangeInterval int) *Term {
	return &Term{
		id:                 id,
		flag:               false,
		partsNum:           participantNumber,
		termChangeInterval: termChangeInterval,
		senators:           make(Senators),
		formerSenators:     make(map[uint32]Senators),
		candidates:         make(map[common.Address]*campaign.Campaign),
		alsorans:           make(map[common.Address]*campaign.Campaign),
		campaigns:          make(map[common.Address]*campaign.Campaign),
	}
}

func (t *Term) ID() uint32 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.id
}

func (t *Term) UpdateID(id uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.id = id
}

func (t *Term) SwitchFlag(flag bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flag = flag
}

func (t *Term) SetStartedHeight(h uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.startedHeight = h
}

func (t *Term) GetGenesisTermChange() *campaign.TermChange {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.genesisTermChange
}

func (t *Term) Changing() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.flag
}

func (t *Term) Started() bool {
	return t.started
}

func (t *Term) GetCandidate(addr common.Address) *campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.candidates[addr]
}

func (t *Term) Candidates() map[common.Address]*campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.candidates
}

func (t *Term) AddCandidate(c *campaign.Campaign, publicKey crypto.PublicKey) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.candidates[c.Sender()] = c
	t.publicKeys = append(t.publicKeys, publicKey)
	//sort.Sort(t.publicKeys)
}

func (t *Term) AddCampaign(c *campaign.Campaign) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.campaigns[c.Sender()] = c
}

func (t *Term) GetCampaign(addr common.Address) *campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.campaigns[addr]
}

func (t *Term) Campaigns() map[common.Address]*campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.campaigns
}

func (t *Term) GetAlsoran(addr common.Address) *campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.alsorans[addr]
}

func (t *Term) Alsorans() map[common.Address]*campaign.Campaign {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.alsorans
}

func (t *Term) AddAlsorans(camps []*campaign.Campaign) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, c := range camps {
		if c == nil {
			continue
		}
		// TODO
		// this check is not proper enough, try optimize it.
		if t.hasCampaign(c.Sender()) {
			continue
		}
		t.alsorans[c.Sender()] = c
	}
}

func (t *Term) HasCampaign(address common.Address) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.hasCampaign(address)
}

func (t *Term) hasCampaign(address common.Address) bool {
	if _, exists := t.candidates[address]; exists {
		log.Debug("exist in candidates")
		return true
	}
	if _, exists := t.campaigns[address]; exists {
		log.Debug("exist in campaigns")
		return true
	}
	if _, exists := t.alsorans[address]; exists {
		log.Debug("exist in alsorans")
		return true
	}
	return false
}

// CanChange returns true if the campaigns cached reaches the
// term change requirments.
func (t *Term) CanChange(lastHeight uint64, isGenesis bool) bool {
	//TODO change this in future , make more slower
	var a = 1
	var b = 0
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO:
	// term change requirements are not enough now.
	if len(t.campaigns) == 0 {
		return false
	}
	if len(t.campaigns) < t.partsNum {
		log.WithField("len ", len(t.campaigns)).Debug("not enough campaigns , waiting")
		return false
	}
	if isGenesis {
		log.Debug("is genesis consensus, change term")
		return true
	}
	if lastHeight-t.startedHeight < uint64(t.termChangeInterval*a+b) {
		return false
	}
	return true
}

func (t *Term) ChangeTerm(tc *campaign.TermChange, lastHeight uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	snts := make(map[common.Address]*Senator)
	for addr, c := range t.candidates {
		s := newSenator(addr, c.PublicKey, tc.PkBls)
		snts[addr] = s
	}

	t.formerPublicKeys = t.publicKeys
	if t.id == 0 {
		t.genesisTermChange = tc
	}

	t.currentTermChange = tc

	t.candidates = make(map[common.Address]*campaign.Campaign)
	t.alsorans = make(map[common.Address]*campaign.Campaign)
	t.campaigns = make(map[common.Address]*campaign.Campaign)
	t.publicKeys = nil

	formerSnts := t.senators
	t.formerSenators[t.id] = formerSnts

	t.senators = snts
	t.started = true

	// TODO
	// 1. update id.
	// 2. process alsorans.

	t.id++
	t.startedHeight = lastHeight
	log.WithField("startedHeight", t.startedHeight).WithField("len senators ", len(t.senators)).WithField("id ", t.id).Info("term changed , id updated")
	t.flag = false
	return nil
}

type Senator struct {
	addr         common.Address
	pk           []byte
	blspk        []byte
	Id           int
	CampaignHash common.Hash
	// TODO:
	// more variables?
}

func newSenator(addr common.Address, publickey, blspk []byte) *Senator {
	s := &Senator{}
	s.addr = addr
	s.pk = publickey
	s.blspk = blspk

	return s
}

type Senators map[common.Address]*Senator

func (t *Term) GetSenator(address common.Address) *Senator {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if v, ok := t.senators[address]; ok {
		return v
	}
	return nil
}

func (t *Term) ClearCampaigns() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.campaigns = nil
}

func (t *Term) GetFormerPks() []crypto.PublicKey {
	return t.formerPublicKeys
}
