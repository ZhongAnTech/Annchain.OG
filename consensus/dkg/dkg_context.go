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
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/annchain/kyber/v3/share"
	"github.com/annchain/kyber/v3/share/dkg/pedersen"
	"github.com/annchain/kyber/v3/sign/bls"
	"github.com/annchain/kyber/v3/sign/tbls"
)

// DkgContext stores the DKG info collected from other peers.
// It can sign messages individually and recover the joint sig once enough peers share their partSigs
// It is the core algorithm of DKG
type DkgContext struct {
	SessionId uint32
	MyIndex   uint32
	Me        PartSec
	PartPubs  []PartPub // must be ordered from the outside

	//CandidatePartSec      []kyber.Scalar
	//CandidatePublicKey    [][]byte
	//addressIndex          map[common.Address]int
	//SecretKeyContribution map[common.Address]kyber.Scalar
	Suite *bn256.Suite
	Dkger *dkg.DistKeyGenerator // backend algorithm
	//Resps                 map[common.Address]*dkg.Response
	//dealsIndex            map[uint32]bool
	Threshold      int
	NbParticipants int
	JointPubKey    kyber.Point
	//responseNumber        int
	//SigShares [][]byte
	KeyShare *dkg.DistKeyShare // cache of the DistKeyShare to avoid recovery multiple times
}

func NewDkgContext(s *bn256.Suite, termId uint32) *DkgContext {
	c := &DkgContext{
		Suite: s,
		//addressIndex:          make(map[common.Address]int),
		//SecretKeyContribution: make(map[common.Address]kyber.Scalar),
		//Resps:                 make(map[common.Address]*dkg.Response),
		SessionId: termId,
		//dealsIndex:            make(map[uint32]bool),
	}
	return c
}

// GenerateDKGer inits a dkg by all part-public keys and my part-private key
// Then it is possible to sign and verify data.
func (p *DkgContext) GenerateDKGer() error {
	// use all partPubs and my partSec to generate a dkg
	log.WithField("partPubLen", len(p.PartPubs)).Debug("generating DKGer")
	participants := PartPubs(p.PartPubs).Points()
	dkger, err := dkg.NewDistKeyGenerator(p.Suite, p.Me.Scalar, participants, p.Threshold)
	if err != nil {
		log.WithField("dkger", dkger).WithError(err).Error("generate dkg error")
		return err
	}
	p.Dkger = dkger
	p.KeyShare = nil
	return nil
}

func (p *DkgContext) ensureKeyShare() (err error) {
	dks := p.KeyShare
	if dks == nil {
		dks, err = p.Dkger.DistKeyShare()
		if err != nil {
			return
		}
		p.KeyShare = dks
	}
	return nil
}

// VerifyByPubPoly verifies signature for msg
func (p *DkgContext) VerifyByPubPoly(msg []byte, sig []byte) (err error) {
	if err = p.ensureKeyShare(); err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), p.KeyShare.Commitments())

	//TODO： remove these check. It is only for debugging
	if pubPoly.Commit() != p.KeyShare.Public() {
		err = errors.New("PubPoly not aligned to dksPublic")
		return
	}

	err = bls.Verify(p.Suite, pubPoly.Commit(), msg, sig)
	log.Tracef(" pubPolyCommit [%s] dksPublic [%s] dksCommitments [%s]\n",
		pubPoly.Commit(), p.KeyShare.Public(), p.KeyShare.Commitments())
	return
}

// VerifyByDksPublic verifies signature for msg
func (p *DkgContext) VerifyByDksPublic(msg []byte, sig []byte) (err error) {
	if err = p.ensureKeyShare(); err != nil {
		return
	}
	err = bls.Verify(p.Suite, p.KeyShare.Public(), msg, sig)
	return
}

// RecoverSig builds a jointSignature from sigShares collected from enough participants
func (p *DkgContext) RecoverSig(msg []byte, sigShares [][]byte) (jointSig []byte, err error) {
	if err = p.ensureKeyShare(); err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), p.KeyShare.Commitments())
	jointSig, err = tbls.Recover(p.Suite, pubPoly, msg, sigShares, p.Threshold, p.NbParticipants)
	return
}

// RecoverPub builds a joint public key from keyshares collected from all other participants
func (p *DkgContext) RecoverPub() (jointPubKey kyber.Point, err error) {
	if err = p.ensureKeyShare(); err != nil {
		return
	}
	pubPoly := share.NewPubPoly(p.Suite, p.Suite.Point().Base(), p.KeyShare.Commitments())
	jointPubKey = pubPoly.Commit()
	p.JointPubKey = jointPubKey
	return
}

// PartSig signs the message into a single sigShare.
func (p *DkgContext) PartSig(msg []byte) (partSig []byte, err error) {
	if err = p.ensureKeyShare(); err != nil {
		return
	}
	partSig, err = tbls.Sign(p.Suite, p.KeyShare.PriShare(), msg)
	return
}
