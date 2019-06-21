package bit_test

import (
	"fmt"
	"github.com/annchain/OG/types"
	"testing"
)

func TestBits(t *testing.T) {

	b := types.RandomHash().ToBytes()
	k := 4

	//for j=:0;j<100;j++ {
	//
	//}
	mask := byte(1 << uint(k-1))
	p := b[2] & mask
	if p > byte(0) {
		fmt.Println(true)
	} else {
		fmt.Println(false)
	}
	fmt.Printf("\n%b  %b %b", b[2], p, mask)
}
