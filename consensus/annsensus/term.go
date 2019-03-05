package annsensus

import (
	"sync"

	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type Term struct {
	id            uint64
	flag          bool
	partsNum      int
	senators      Senators
	formerSentors map[uint64]Senators
	candidates    map[types.Address]*types.Campaign
	alsorans      map[types.Address]*types.Campaign

	mu sync.RWMutex
}

func newTerm(id uint64, pn int) *Term {
	return &Term{
		id:            id,
		flag:          false,
		partsNum:      pn,
		senators:      make(Senators),
		formerSentors: make(map[uint64]Senators),
		candidates:    make(map[types.Address]*types.Campaign),
		alsorans:      make(map[types.Address]*types.Campaign),
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
	_, ok := t.candidates[c.Issuer]
	if !ok {
		_, ok = t.alsorans[c.Issuer]
	}
	return ok
}

// CanChange returns true if the campaigns cached reaches the
// term change requirments.
func (t *Term) CanChange() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO
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
	// TODO
}

func newSenator(addr types.Address, publickey, blspk []byte) *Senator {
	s := &Senator{}
	s.addr = addr
	s.pk = publickey
	s.blspk = blspk

	return s
}

type Senators map[types.Address]*Senator
