package txmaker

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
)

// FIFOTipGenerator wraps txpool random picker so that
// when tps is low, the graph will not be like a chain.
type FIFOTipGenerator struct {
	maxCacheSize int
	upstream     TipGenerator
	fifoRing     []ogmessage.Txi
	fifoRingPos  int
	fifoRingFull bool
	mu           sync.RWMutex
}

func NewFIFOTIpGenerator(upstream TipGenerator, maxCacheSize int) *FIFOTipGenerator {
	return &FIFOTipGenerator{
		upstream:     upstream,
		maxCacheSize: maxCacheSize,
		fifoRing:     make([]ogmessage.Txi, maxCacheSize),
	}
}

func (f *FIFOTipGenerator) validation() {
	var k int
	for i := 0; i < f.maxCacheSize; i++ {
		if f.fifoRing[i] == nil {
			if f.fifoRingPos > i {
				f.fifoRingPos = i
			}
			break
		}
		//if the tx is became fatal tx ,remove it from tips
		if f.fifoRing[i].InValid() {
			logrus.WithField("tx ", f.fifoRing[i]).Debug("found invalid tx")
			if !f.fifoRingFull {
				if f.fifoRingPos > 0 {
					if f.fifoRingPos-1 == i {
						f.fifoRing[i] = nil
					} else if f.fifoRingPos-1 > i {
						f.fifoRing[i] = f.fifoRing[f.fifoRingPos-1]
					} else {
						f.fifoRing[i] = nil
						break
					}
					f.fifoRingPos--
				}
			} else {
				f.fifoRing[i] = f.fifoRing[f.maxCacheSize-k-1]
				f.fifoRing[f.maxCacheSize-k-1] = nil
				i--
				k++
				f.fifoRingFull = false
			}
		}
	}
}

func (f *FIFOTipGenerator) GetByNonce(addr common.Address, nonce uint64) ogmessage.Txi {
	return f.upstream.GetByNonce(addr, nonce)
}

func (f *FIFOTipGenerator) IsBadSeq(seq *ogmessage.Sequencer) error {
	return f.upstream.IsBadSeq(seq)
}

func (f *FIFOTipGenerator) GetRandomTips(n int) (v []ogmessage.Txi) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upstreamTips := f.upstream.GetRandomTips(n)
	//checkValidation
	f.validation()
	// update fifoRing
	for _, upstreamTip := range upstreamTips {
		duplicated := false
		for i := 0; i < f.maxCacheSize; i++ {
			if f.fifoRing[i] != nil && f.fifoRing[i].GetTxHash() == upstreamTip.GetTxHash() {
				// duplicate, ignore directly
				duplicated = true
				break
			}
		}
		if !duplicated {
			// advance ring and put it in the fifo ring
			f.fifoRing[f.fifoRingPos] = upstreamTip
			f.fifoRingPos++
			// round
			if f.fifoRingPos == f.maxCacheSize {
				f.fifoRingPos = 0
				f.fifoRingFull = true
			}
		}
	}
	// randomly pick n from fifo cache
	randIndices := make(map[int]bool)
	ringSize := f.fifoRingPos
	if f.fifoRingFull {
		ringSize = f.maxCacheSize
	}
	pickSize := n
	if !f.fifoRingFull && f.fifoRingPos < n {
		pickSize = f.fifoRingPos
	}

	for len(randIndices) != pickSize {
		randIndices[rand.Intn(ringSize)] = true
	}

	// dump those txs
	for k := range randIndices {
		v = append(v, f.fifoRing[k])
	}
	return

}
