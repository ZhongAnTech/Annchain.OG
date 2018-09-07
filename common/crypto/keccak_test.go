package crypto

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/math"
	"testing"
)

func TestFromECDSA(t *testing.T) {
	priv, err := GenerateKey()
	fmt.Println(err, priv)
	data := math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	fmt.Println(hex.EncodeToString(data))
}
