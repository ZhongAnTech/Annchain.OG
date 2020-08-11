package ogsyncer

import (
	"github.com/annchain/commongo/math"
	"sync"
	"time"
)

type HeightInfo struct {
	Height      int64
	UpdatedTime time.Time
}

type PeerManager struct {
	peerHeights        map[string]HeightInfo
	knownMaxPeerHeight int64
	myHeight           int64
	mu                 sync.RWMutex
}

func (b *PeerManager) updateKnownPeerHeight(peerId string, height int64) (refreshed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// refresh our target
	refreshed = height > b.knownMaxPeerHeight
	b.knownMaxPeerHeight = math.BiggerInt64(b.knownMaxPeerHeight, height)
	b.peerHeights[peerId] = HeightInfo{
		Height:      height,
		UpdatedTime: time.Now(),
	}
	return
}

func (b *PeerManager) removeKnownPeerHeight(peerId string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// refresh our target
	delete(b.peerHeights, peerId)
}

func (b *PeerManager) getAllHigherPeers(height int64) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	higherPeers := []string{}
	for peerId, heightInfo := range b.peerHeights {
		if heightInfo.Height >= height {
			higherPeers = append(higherPeers, peerId)
		}
	}
	return higherPeers
}

func (b *PeerManager) findOutdatedPeersToQueryHeight(limit int) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	needUpdatePeers := []string{}

	oldTime := time.Now().Add(time.Minute * -1) // query height for every 1 minute

	for peerId, heightInfo := range b.peerHeights {
		if heightInfo.UpdatedTime.Before(oldTime) {
			needUpdatePeers = append(needUpdatePeers, peerId)
			if len(needUpdatePeers) >= limit {
				break
			}
		}
	}
	return needUpdatePeers
}
