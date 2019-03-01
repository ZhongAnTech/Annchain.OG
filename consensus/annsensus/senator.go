package annsensus

import (
	"github.com/annchain/OG/types"
)

type Senator struct {
	addr  types.Address
	pk    []byte
	blspk []byte
	// TODO
}

func newSenator(addr types.Address, publickey, blspk []byte) *Senator {
	s := &Senator{}
	s.addr = addr
	s.pk = publickey
	s.blspk = blspk

	return s
}
