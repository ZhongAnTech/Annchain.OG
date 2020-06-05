package syncer

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"testing"
)

func TestSequencerCache_Add(t *testing.T) {
	p := NewSequencerCache(7)
	fmt.Println(p)
	var hashes types.Hashes
	for i := 0; i < 10; i++ {
		hash := types.RandomHash()
		hashes = append(hashes, hash)
		p.Add(hash, fmt.Sprintf("%d", i))
	}
	fmt.Println(hashes)
	fmt.Println(p)
	for _, hash := range hashes {
		fmt.Println(p.GetPeer(hash), hash)
	}
}
