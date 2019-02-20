package annsensus


import "github.com/annchain/OG/types"

// genCamp calculate vrf and generate a campaign that contains this vrf info.
func (as *AnnSensus) genCamp() *types.Campaign {
	//once for test
	//as.doCamp = false
	// TODO
	base := types.TxBase{
		Type:types.TxBaseTypeCampaign,
	}

	cp:=  &types.Campaign{
		TxBase:base,
	}
	if !as.GenerateVrf(cp){
		return nil
	}
	pubKey := as.GenerateDKgPublicKey()
	cp.Vrf.PublicKey = pubKey
	return cp
}
