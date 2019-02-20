package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/poc/dkg"
	"github.com/sirupsen/logrus"
)

func (as *AnnSensus) GenerateDKgPublicKey()( dkgPubkey []byte){
	s := bn256.NewSuiteG2()
	partner := & dkg.Partner{
		Suite:          s,
	}
	// partner generate pair for itself.
	partSec, partPub := genPartnerPair(partner)
	// you should always keep MyPartSec safe. Do not share.
	partner.MyPartSec = partSec
	as.partNer = partner
	as.partNer.NbParticipants = as.NbParticipants
	as.partNer.Threshold = as.Threshold
	fmt.Println(partPub)
	dkgPubkey, err:= partPub.MarshalBinary()
	if err!=nil {
		logrus.WithError(err).Error("marshal public key error")
		panic(err)
		return nil
	}
	return dkgPubkey

}


func genPartnerPair(p * dkg.Partner) (kyber.Scalar, kyber.Point) {
	sc := p.Suite.Scalar().Pick(p.Suite.RandomStream())
	return sc, p.Suite.Point().Mul(sc, nil)
}