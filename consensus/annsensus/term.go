package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"sync"
)

type HeightInfoCarrier interface {
	ProvideHeight() uint64
}

type TermCollection struct {
	contextProvider ConsensusContext
	BftPartner      bft.BftPartner
	DkgPartner      dkg.DkgPartner
	quit            chan bool
	quitWg          sync.WaitGroup
}

func NewTermCollection(
	contextProvider ConsensusContext,
	bftPartner bft.BftPartner, dkgPartner dkg.DkgPartner) *TermCollection {
	return &TermCollection{
		contextProvider: contextProvider,
		BftPartner:      bftPartner,
		DkgPartner:      dkgPartner,
		quit:            make(chan bool),
		quitWg:          sync.WaitGroup{},
	}
}

func (tc *TermCollection) Start() {
	// start all operators for this term.
	tc.quitWg.Add(1)
loop:
	for {
		select {
		case <-tc.quit:
			tc.BftPartner.Stop()
			tc.DkgPartner.Stop()
			tc.quitWg.Done()
			break loop
		}
	}
}

func (tc *TermCollection) Stop() {
	close(tc.quit)
	tc.quitWg.Wait()
}


type AnnsensusTermHolder struct {
	// in case of disordered message, cache the terms and the correspondent processors.
	// TODO: wipe it constantly
	termMap      map[uint32]*TermCollection
	termProvider TermIdProvider
	mu           sync.RWMutex
	debugMyId    int
}

func (b *AnnsensusTermHolder) GetTermById(termId uint32) (msgTerm *TermCollection, ok bool) {
	msgTerm, ok = b.termMap[termId]
	return
}

func (b *AnnsensusTermHolder) SetTerm(termId uint32, termCollection *TermCollection) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.termMap[termId] = termCollection
	b.debugMyId = termCollection.contextProvider.GetMyBftId()
}

func NewAnnsensusTermHolder(termProvider TermIdProvider) *AnnsensusTermHolder {
	return &AnnsensusTermHolder{
		termProvider: termProvider,
		termMap:      make(map[uint32]*TermCollection),
	}
}

func (b *AnnsensusTermHolder) GetTermByHeight(heightInfoCarrier HeightInfoCarrier) (msgTerm *TermCollection, ok bool) {
	height := heightInfoCarrier.ProvideHeight()
	// judge term
	termId := b.termProvider.HeightTerm(height)

	b.mu.RLock()
	defer b.mu.RUnlock()
	msgTerm, ok = b.termMap[termId]
	return
}

func (b *AnnsensusTermHolder) DebugMyId() int {
	return b.debugMyId
}
