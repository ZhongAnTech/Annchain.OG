package annsensus

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAnnSensus_VrfVerify(t *testing.T) {
	for i := 0; i < 100; i++ {
		a := &AnnSensus{}
		cp := &types.Campaign{}
		a.Idag = &DummyDag{}
		a.doCamp = true
		vrf := a.GenerateVrf()
		fmt.Println(vrf, i)
		if vrf == nil {
			continue
		}
		cp.Vrf = *vrf
		err := a.VrfVerify(cp.Vrf.Vrf, cp.Vrf.PublicKey, cp.Vrf.Message, cp.Vrf.Proof)
		require.Nil(t, err)
		return
	}

}
