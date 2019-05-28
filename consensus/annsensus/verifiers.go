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
	"bytes"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/types"
	"time"
)

// consensus related verification
func (a *AnnSensus) VerifyTermChange(t *types.TermChange) bool {
	if a.disable {
		log.WithField("t ", t).Warn("annsensus disabled ")
		return true
	}
	//check balance
	if t.TermID <= a.term.ID() {
		//small term id senstors will be dropped , just verify format and bls keys
		log.Debug("small term id ")
	} else {
		if a.GetCandidate(t.Issuer) == nil {
			log.WithField("candidates ", a.term.candidates).WithField("addr ", t.Issuer.TerminalString()).Warn("not found campaign for termChange")
			return false
		}
	}
	if len(t.SigSet) < a.dkg.partner.NbParticipants {
		log.WithField("len ", len(t.SigSet)).WithField("need ",
			a.dkg.partner.NbParticipants).Warn("not eoungh sigsets")
		return false
	}
	signer := crypto.NewSigner(a.cryptoType)
	for _, sig := range t.SigSet {
		if sig == nil {
			log.Warn("nil sig")
			return false
		}
		pk := crypto.PublicKeyFromBytes(a.cryptoType, sig.PublicKey)
		if !signer.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, sig.Signature), t.PkBls) {
			log.WithField("sig ", sig).Warn("Verify Signature for sigsets fail")
			return false
		}
	}
	log.WithField("tc ", t).Trace("verify ok ")
	return true

}

// consensus related verification
func (a *AnnSensus) VerifySequencer(seq *types.Sequencer) bool {

	if a.disable {
		log.WithField("seq ", seq).Warn("annsensus disabled ")
		return true
	}

	if senator := a.term.GetSenator(seq.Issuer); senator == nil {
		log.WithField("address ", seq.Issuer.ShortString()).Warn("not found senator for address")
		return false
	}
	if seq.Proposing {
		log.WithField("hash ", seq).Debug("proposing seq")
		return true
	}
	log.WithField("hash ", seq).Debug("normal seq")

	if !a.term.started {
		if seq.Height > 1 {
			log.Warn("dkg is not ready yet")
			return false
		}
		//wait 2 second
		for {
			select {
			case <-time.After(50 * time.Millisecond):
				if a.term.started {
					break
				}
			case <-a.close:
				return false
			case <-time.After(2 * time.Second):
				log.Warn("dkg is not ready yet")
				return false
			}
		}

	}
	ok := a.dkg.VerifyBlsSig(seq.GetTxHash().ToBytes(), seq.BlsJointSig, seq.BlsJointPubKey, a.bft.DKGTermId)

	if !ok {
		return false
	}
	//TODO more consensus verifications
	return true
}

// consensus related verification
func (a *AnnSensus) VerifyCampaign(cp *types.Campaign) bool {

	if a.disable {
		log.WithField("cp ", cp).Warn("annsensus disabled ")
		return true
	}

	//check balance
	balance := a.Idag.GetBalance(cp.Issuer)
	if balance.Value.Cmp(campaigningMinBalance.Value) < 0 {
		log.WithField("addr ", cp.Issuer).Warn("your balance is not enough to generate campaign")
		return false
	}

	err := a.VrfVerify(cp.Vrf.Vrf, cp.Vrf.PublicKey, cp.Vrf.Message, cp.Vrf.Proof)
	if err != nil {
		log.WithError(err).Debug("vrf verify failed")
		return false
	}

	err = cp.UnmarshalDkgKey(bn256.UnmarshalBinaryPointG2)
	if err != nil {
		log.WithField("cp", cp).WithError(err).Debug("dkg Public key  verify failed")
		return false
	}
	if cp.GetDkgPublicKey() == nil {
		log.WithField("cp", cp).WithField("data ", cp.PublicKey).Warn("dkgPub is nil")
		return false
	}
	if a.HasCampaign(cp.Issuer) {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
		//todo  if a node dose not cacht up yet  returning false will cause fail
		//return false
	}
	log.WithField("cp ", cp).Trace("verify ok ")
	return true
}

func (a *AnnSensus) VerifyRequestedTermChange(t *types.TermChange) bool {

	if a.disable {
		log.WithField("t ", t).Warn("annsensus disabled ")
		return true
	}

	if len(t.SigSet) < a.dkg.partner.NbParticipants {
		log.WithField("len ", len(t.SigSet)).WithField("need ",
			a.dkg.partner.NbParticipants).Warn("not enough sigsets")
		return false
	}
	signer := crypto.NewSigner(a.cryptoType)
	for _, sig := range t.SigSet {
		if sig == nil {
			log.Warn("nil sig")
			return false
		}
		pk := crypto.PublicKeyFromBytes(a.cryptoType, sig.PublicKey)
		var ok bool
		for _, gPk := range a.genesisAccounts {
			if bytes.Equal(pk.Bytes, gPk.Bytes) {
				ok = true
			}
		}
		if !ok {
			log.WithField("pk ", pk).Warn("not a consensus participater")
			return false
		}
		if !signer.Verify(pk, crypto.SignatureFromBytes(a.cryptoType, sig.Signature), t.PkBls) {
			log.WithField("sig ", sig).Warn("Verify Signature for sigsets fail")
			return false
		}
	}
	log.WithField("tc ", t).Trace("verify ok ")
	return true
}
