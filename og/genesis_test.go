package og_test

import (
	"testing"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/types"
)

// TODO
func TestName(t *testing.T) {
	a := types.HexToHash("0x00")
	logrus.Info(a)
}