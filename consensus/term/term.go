package term

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
)

type Senator struct {
	Id           int
	Address      common.Address
	PublicKey    crypto.PublicKey
	BlsPublicKey dkg.PartPub
}

// Term holds all public info about the term.
type Term struct {
	Id                uint32
	PartsNum          int
	Threshold         int
	Senators          []Senator
	AllPartPublicKeys []dkg.PartPub
	PublicKey         kyber.Point
	ActivateHeight    uint64
	Suite             *bn256.Suite
}
