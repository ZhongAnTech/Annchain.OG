package dkg

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/kyber/v3"
)

type PartPub struct {
	kyber.Point
	PublicKey crypto.PublicKey
}

type PartPubs []PartPub

func (p PartPubs) Points() []kyber.Point {
	var poits []kyber.Point
	for _, v := range p {
		poits = append(poits, v.Point)
	}
	return poits
}

func (h PartPubs) Len() int {
	return len(h)
}

func (h PartPubs) Less(i, j int) bool {
	return h[i].PublicKey.String() < h[j].PublicKey.String()
}

func (h PartPubs) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

