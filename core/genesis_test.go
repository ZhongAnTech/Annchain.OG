package core

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestName(t *testing.T) {
	a := types.HexToHash("0x00")
	logrus.Info(a)
	DefaultGenesis(crypto.CryptoTypeSecp256k1)
}
