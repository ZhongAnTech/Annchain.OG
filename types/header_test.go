package types

import (
	"fmt"
	"testing"
)

func TestNewSequencerHead(t *testing.T) {

	var seqs Sequencers
	for i := 0; i < 3; i++ {
		s := Sequencer{
			TxBase: TxBase{
				Hash:   randomHash(),
				Height: uint64(i),
			},
		}
		seqs = append(seqs, &s)
	}
	for i := 0; i < 3; i++ {
		fmt.Println(seqs[i])
		fmt.Println(*seqs[i])
	}

	headres := seqs.ToHeaders()
	for i := 0; i < 3; i++ {
		fmt.Println(headres[i])
		//fmt.Println(*headres[i])
	}
}
