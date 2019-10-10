package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
	"sync"
)

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

func NewBftTermHolder(termProvider TermProvider) *AnnsensusTermHolder {
	return &AnnsensusTermHolder{
		termProvider: termProvider,
		termMap:      make(map[uint32]*TermCollection),
	}
}

func (b *AnnsensusTermHolder) GetTermCollection(ogMessage *message.OGMessage) (msgTerm *TermCollection, err error) {
	height := ogMessage.Message.(*bft.BftBasicInfo).HeightRound.Height
	// judge term
	termId := b.termProvider.HeightTerm(height)
	msgTerm, ok := b.termMap[termId]
	if !ok {
		err = errors.New("term not found while handling bft message")
	}
	return
}
