package annsensus

import (
	"github.com/annchain/OG/types"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	log "github.com/sirupsen/logrus"
)

// consensus related verification
func (a *AnnSensus) VerifyTermChange(t *types.TermChange) bool {
	//check balance
	if a.GetCandidate(t.Issuer) == nil {
		log.WithField("addr ", t.Issuer.TerminalString()).Warn("not found  campaign for tearmchange")
		return false
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
		pk := crypto.PublicKeyFromBytes(CryptoType, sig.PublicKey)
		if !signer.Verify(pk, crypto.SignatureFromBytes(CryptoType, sig.Signature), t.PkBls) {
			log.WithField("sig ", sig).Warn("Verify Signature for sigsets fail")
			return false
		}
	}
	log.WithField("tc ", t).Trace("verify ok ")
	return true
}

// consensus related verification
func (a *AnnSensus) VerifySequencer(seq *types.Sequencer) bool {
	return true
}

// consensus related verification
func (a *AnnSensus) VerifyCampaign(cp *types.Campaign) bool {
	//check balance
	balance := a.Idag.GetBalance(cp.Issuer)
	if balance.Value.Cmp(campaigningMinBalance.Value) < 0 {
		log.Warn("your balance is not enough to generate campaign")
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
	if a.HasCampaign(cp) {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
		return false
	}
	log.WithField("cp ", cp).Trace("verify ok ")
	return true
}
