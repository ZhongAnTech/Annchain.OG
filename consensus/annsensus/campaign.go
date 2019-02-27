package annsensus

import (
	"fmt"

	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

var campaigningMinBalance = math.NewBigInt(1000)

// genCamp calculate vrf and generate a campaign that contains this vrf info.
func (as *AnnSensus) genCamp(dkgPub []byte) *types.Campaign {
	//once for test
	//as.doCamp = false
	// TODO
	base := types.TxBase{
		Type: types.TxBaseTypeCampaign,
	}
	address := as.MyPrivKey.PublicKey().Address()
	balance := as.Idag.GetBalance(address)
	if balance.Value.Cmp(campaigningMinBalance.Value) < 0 {
		log.Debug("not enough balance to gen campaign")
		return nil
	}
	cp := &types.Campaign{
		TxBase: base,
	}
	if vrf := as.GenerateVrf(); vrf != nil {
		cp.Vrf = *vrf
	} else {
		return nil
	}
	//cp.Vrf.PublicKey = pub TODO why
	cp.DkgPublicKey = dkgPub
	return cp
}

//consensus related verification
func (a *AnnSensus) VerifyTermChange(t *types.TermChange) bool {
	return true
}

//consensus related verification
func (a *AnnSensus) VerifySequencer(seq *types.Sequencer) bool {
	return true
}

//consensus related verification
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
		log.WithError(err).Debug("dkg Public key  verify failed")
		return false
	}
	if a.HasCampaign(cp) {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
		return false
	}
	return true
}

//ProcessCampaign  , verify campaign
func (a *AnnSensus) AddCampaignCandidates(cp *types.Campaign) error {
	if a.HasCampaign(cp) {
		log.WithField("campaign", cp).Debug("duplicate campaign ")
		return fmt.Errorf("duplicate ")
	}
	a.partner.PartPubs = append(a.partner.PartPubs, cp.GetDkgPublicKey())
	a.candidates[cp.Issuer] = cp
	a.partner.adressIndex[cp.Issuer] = len(a.partner.PartPubs) - 1

	return nil
}

//ProcessTermChange , verify termchange
func (a *AnnSensus) ProcessTermChange(tc *types.TermChange) error {
	return nil
}

func (a *AnnSensus) HasCampaign(cp *types.Campaign) bool {
	_, ok := a.candidates[cp.Issuer]
	if !ok {
		_, ok = a.alsorans[cp.Issuer]
	}
	return ok
}
