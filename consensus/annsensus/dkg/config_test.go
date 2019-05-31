package dkg

import (
	"fmt"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/share"
	"reflect"

	"go.dedis.ch/kyber/v3/share/dkg/pedersen"
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
	d := NewDkg(true, 21, 15, nil, nil, nil, nil)
	d.ConfigFilePath = "test.json"
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
	fmt.Println(d.generateConfig())
	d.SaveConsensusData()
	config, _ := d.LoadConsensusData()
	fmt.Println(config)
}
