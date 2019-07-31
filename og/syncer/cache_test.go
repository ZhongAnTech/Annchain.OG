package syncer

import (
	"fmt"
	"github.com/annchain/OG/common"
	"testing"
)

func TestSequencerCache_Add(t *testing.T) {
	p:= NewSequencerCache(7)
	fmt.Println(p)
	var hashes common.Hashes
	for i:=0;i<10;i++ {
		hash := common.RandomHash()
		hashes = append(hashes,hash)
		p.Add(hash, fmt.Sprintf("%d",i))
	}
	fmt.Println(hashes)
	fmt.Println(p)
	for _,hash := range hashes {
		fmt.Println(p.GetPeer(hash),hash)
	}
}
