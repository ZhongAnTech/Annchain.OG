// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package dkg

import (
	"fmt"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/annchain/kyber/v3/share"
	"reflect"

	"github.com/annchain/kyber/v3/share/dkg/pedersen"
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
