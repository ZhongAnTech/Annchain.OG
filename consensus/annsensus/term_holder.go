package annsensus

import (
	"errors"
	"sync"
)

type HeightInfoCarrier interface {
	ProvideHeight() uint64
}

type AnnsensusTermHolder struct {
	// in case of disordered message, cache the terms and the correspondent processors.
	// TODO: wipe it constantly
	termMap      map[uint32]*TermCollection
	termProvider TermProvider
	mu           sync.Mutex
}

func (b *AnnsensusTermHolder) SetTerm(termId uint32, termCollection *TermCollection) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.termMap[termId] = termCollection
}

func NewAnnsensusTermHolder(termProvider TermProvider) *AnnsensusTermHolder {
	return &AnnsensusTermHolder{
		termProvider: termProvider,
		termMap:      make(map[uint32]*TermCollection),
	}
}

func (b *AnnsensusTermHolder) GetTermCollection(heightInfoCarrier HeightInfoCarrier) (msgTerm *TermCollection, err error) {
	height := heightInfoCarrier.ProvideHeight()
	// judge term
	termId := b.termProvider.HeightTerm(height)
	msgTerm, ok := b.termMap[termId]
	if !ok {
		err = errors.New("term not found")
	}
	return
}
