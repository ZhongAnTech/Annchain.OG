package og

import (
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/deckarep/golang-set"
	"testing"
)

func TestPeerSet_GetRandomPeers(t *testing.T) {

	set := newPeerSet()
	for i := 0; i < 10; i++ {
		rawPeer := p2p.Peer{}
		p := &peer{
			Peer:      &rawPeer,
			version:   1,
			id:        fmt.Sprintf("%d", i),
			knownMsg:  mapset.NewSet(),
			queuedMsg: make(chan []*P2PMessage, maxqueuedMsg),
			term:      make(chan struct{}),
		}
		set.Register(p)
	}
	fmt.Println("len peers", set.Len())
	peers := set.GetRandomPeers(3)
	if len(peers) != 3 {
		t.Fatalf("peers size mismatch, wanted 3 ,got %d ,peers %v ", len(peers), peers)
	}
	peers = set.GetPeers(nil, 3)
	if len(peers) != 0 {
		t.Fatalf("peers size mismatch, wanted 3 ,got %d ,peers %v", len(peers), peers)
	}

}
