package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"testing"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
)

func TestAnnSensus_GenerateDKgPublicKey(t *testing.T) {
	var as AnnSensus
	pk := as.GenerateDKgPublicKey()
	fmt.Println(hexutil.Encode(pk))
	point ,err := bn256.UnmarshalBinaryPointG2(pk)
	if err!=nil {
		t.Fatal(err)
	}
	fmt.Println(point)
}
