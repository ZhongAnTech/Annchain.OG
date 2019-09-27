package annsensus

import (
	"errors"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/og/message"
)

type BftTermHolder struct {
	// in case of disordered message, cache the terms and the correspondent processors.
	// TODO: wipe it constantly
	termMap      map[uint32]*TermComposer
	termProvider TermProvider
}

func NewBftTermHolder(termProvider TermProvider) *BftTermHolder {
	return &BftTermHolder{
		termProvider: termProvider,
		termMap:      make(map[uint32]*TermComposer),
	}
}

func (b *BftTermHolder) GetBftTerm(ogMessage *message.OGMessage) (msgTerm *TermComposer, err error) {
	height := ogMessage.Message.(*bft.BftBasicInfo).HeightRound.Height
	// judge term
	termId := b.termProvider.HeightTerm(height)
	msgTerm, ok := b.termMap[termId]
	if !ok {
		err = errors.New("term not found while handling bft message")
	}
	return
}
