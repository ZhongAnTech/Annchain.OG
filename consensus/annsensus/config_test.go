package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share"
	"reflect"

	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"testing"
)

func genScaler(p *bn256.Suite) kyber.Scalar {
	return p.Scalar().Pick(p.RandomStream())

}

func genPoint(p *bn256.Suite) kyber.Point {
	return p.Point().Mul(p.Scalar().Pick(p.RandomStream()), nil)
}

func TestAnnSensus_SaveConsensusData(t *testing.T) {
	logInit()
	ann := AnnSensus{
		ConfigFilePath: "test.json",
	}
	d := newDkg(&ann, true, 21, 15)
	ann.dkg = d
	d.GenerateDkg()
	suite := bn256.NewSuiteG2()
	fmt.Println(reflect.TypeOf(genScaler(suite)))
	var points []kyber.Point
	points = append(points, genPoint(suite), genPoint(suite))
	var scalars []kyber.Scalar
	scalars = append(scalars, genScaler(suite), genScaler(suite), genScaler(suite))
	d.partner.KeyShare = &dkg.DistKeyShare{
		Commits: points,
		Share: &share.PriShare{
			I: 3,
			V: bn256.NewSuiteG2().Scalar(),
		},
		PrivatePoly: scalars,
	}
	d.partner.jointPubKey = genPoint(suite)
	d.partner.MyPartSec = genScaler(suite)
	fmt.Println(ann.generateConfig())
	ann.SaveConsensusData()
	config, _ := ann.LoadConsensusData()
	fmt.Println(config)
}
