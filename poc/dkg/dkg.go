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
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/bls"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/tbls"
)

type Partner struct {
	ID                    int
	PartPubs              []kyber.Point
	MyPartSec             kyber.Scalar
	SecretKeyContribution []kyber.Scalar
	Suite                 *bn256.Suite
	Dkger                 *dkg.DistKeyGenerator
	Resps                 []*dkg.Response
	Threshold             int
	NbParticipants        int
	SigShares             [][]byte
}

func (p *Partner) GenerateDKGer() *dkg.DistKeyGenerator {
	// use all partPubs and my partSec to generate a dkg
	dkger, err := dkg.NewDistKeyGenerator(p.Suite, p.MyPartSec, p.PartPubs, p.Threshold)
	if err != nil {
		panic(err)
	}
	p.Dkger = dkger
	return dkger
}

func (p *Partner) VerifyByPubPoly(msg []byte, sig []byte) (err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	if pubPoly.Commit() != dks.Public() {
		err = errors.New("PubPoly not aligned to dksPublic")
		return
	}

	err = bls.Verify(p.Suite, pubPoly.Commit(), msg, sig)
	fmt.Printf("ID [%d] pubPolyCommit [%s] dksPublic [%s] dksCommitments [%s]\n",
		p.ID, pubPoly.Commit(), dks.Public(), dks.Commitments())
	return
}

func (p *Partner) VerifyByDksPublic(msg []byte, sig []byte) (err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	err = bls.Verify(p.Suite, dks.Public(), msg, sig)
	return
}

func (p *Partner) RecoverSig(msg []byte) (jointSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointSig, err = tbls.Recover(p.Suite, pubPoly, msg, p.SigShares, p.Threshold, p.NbParticipants)
	return
}

func (p *Partner) RecoverPub() (jointPubKey kyber.Point, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointPubKey = pubPoly.Commit()
	return
}

func (p *Partner) Sig(msg []byte) (partSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	partSig, err = tbls.Sign(p.Suite, dks.PriShare(), msg)
	return
}
