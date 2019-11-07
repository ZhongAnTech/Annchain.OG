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
	Id                uint32        // term id starts from 0 (genesis)
	PartsNum          int           // participants number
	Threshold         int           // participants number for threshold signature
	Senators          []Senator     // all senators to discuss out a signature
	AllPartPublicKeys []dkg.PartPub // their little public keys
	PublicKey         kyber.Point   // the full public key. Only available after collected all others' dkgs
	ActivateHeight    uint64        // when the term is activated.
	Suite             *bn256.Suite  // curve.
}
