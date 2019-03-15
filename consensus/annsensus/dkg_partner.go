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
package annsensus

import (
	"errors"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/bls"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/sign/tbls"
	"github.com/annchain/OG/types"
)

type DKGPartner struct {
	Id                    uint32
	PartPubs              []kyber.Point
	MyPartSec             kyber.Scalar
	CandidatePartSec      kyber.Scalar
	CandidatePublicKey    []byte
	addressIndex          map[types.Address]int
	SecretKeyContribution map[types.Address]kyber.Scalar
	Suite                 *bn256.Suite
	Dkger                 *dkg.DistKeyGenerator
	Resps                 map[types.Address]*dkg.Response
	dealsIndex            map[uint32]bool
	Threshold             int
	NbParticipants        int
	jointPubKey           kyber.Point
	responseNumber        int
	SigShares             [][]byte
}

func NewDKGPartner(s *bn256.Suite) *DKGPartner {
	return &DKGPartner{
		Suite:                 s,
		addressIndex:          make(map[types.Address]int),
		SecretKeyContribution: make(map[types.Address]kyber.Scalar),
		Resps:                 make(map[types.Address]*dkg.Response),
		dealsIndex:            make(map[uint32]bool),
	}
}

func genPartnerPair(p *DKGPartner) (kyber.Scalar, kyber.Point) {
	sc := p.Suite.Scalar().Pick(p.Suite.RandomStream())
	return sc, p.Suite.Point().Mul(sc, nil)
}

func (p *DKGPartner) GenerateDKGer() error {
	// use all partPubs and my partSec to generate a dkg
	log.WithField(" len ", len(p.PartPubs)).Debug("my part pbus")
	dkger, err := dkg.NewDistKeyGenerator(p.Suite, p.MyPartSec, p.PartPubs, p.Threshold)
	if err != nil {
		log.WithField("dkger ", dkger).WithError(err).Error("generate dkg error")
		return err
	}
	p.Dkger = dkger
	return nil
}

func (p *DKGPartner) VerifyByPubPoly(msg []byte, sig []byte) (err error) {
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
	log.Debugf(" pubPolyCommit [%s] dksPublic [%s] dksCommitments [%s]\n",
		pubPoly.Commit(), dks.Public(), dks.Commitments())
	return
}

func (p *DKGPartner) VerifyByDksPublic(msg []byte, sig []byte) (err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	err = bls.Verify(p.Suite, dks.Public(), msg, sig)
	return
}

func (p *DKGPartner) RecoverSig(msg []byte) (jointSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointSig, err = tbls.Recover(p.Suite, pubPoly, msg, p.SigShares, p.Threshold, p.NbParticipants)
	return
}

func (p *DKGPartner) RecoverPub() (jointPubKey kyber.Point, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), dks.Commitments())
	jointPubKey = pubPoly.Commit()
	p.jointPubKey = jointPubKey
	return
}

func (p *DKGPartner) Sig(msg []byte) (partSig []byte, err error) {
	dks, err := p.Dkger.DistKeyShare()
	if err != nil {
		return
	}
	partSig, err = tbls.Sign(p.Suite, dks.PriShare(), msg)
	return
}
