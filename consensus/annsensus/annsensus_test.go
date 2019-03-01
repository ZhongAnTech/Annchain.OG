package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/hexutil"
	"testing"
)

func TestAnnSensus_GenerateDKgPublicKey(t *testing.T) {
	var as = NewAnnSensus(1,true,5,4)
	pk := as.dkg.pk
	fmt.Println(hexutil.Encode(pk))
	point, err := bn256.UnmarshalBinaryPointG2(pk)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(point)
}
