package tx_types

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp

//msgp:tuple TokenInfo
type TokenInfo struct {
	PublicOffering
	Sender       common.Address
	CurrentValue *math.BigInt
	Destroyed    bool
}

type TokensInfo []*TokenInfo
