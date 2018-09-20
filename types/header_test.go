package types

import (
	"fmt"
	"testing"
)

func TestNewSequencerHead(t *testing.T) {

	var seqs []*Sequencer
	for i := 0; i < 3; i++ {
		s := Sequencer{
			Id: uint64(i),
		}
		seqs = append(seqs, &s)
	}
	for i := 0; i < 3; i++ {
		fmt.Println(seqs[i])
		fmt.Println(*seqs[i])
	}

	headres := SeqsToHeaders(seqs)
	for i := 0; i < 3; i++ {
		fmt.Println(headres[i])
		//fmt.Println(*headres[i])
	}
}
