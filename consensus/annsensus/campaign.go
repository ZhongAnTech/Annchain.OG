package annsensus

import (
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

func (a *AnnSensus) HasCampaign(cp *types.Campaign) bool {
	if a.GetCandidate(cp.Issuer) != nil {
		return true
	}
	if a.GetAlsoran(cp.Issuer) != nil {
		return true
	}
	return false
}
