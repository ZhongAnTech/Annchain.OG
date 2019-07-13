package tx_types

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
)

//go:generate msgp

//msgp:tuple TokenInfo
type TokenInfo struct {
	PublicOffering
	Sender       common.Address `json:"sender"`
	CurrentValue *math.BigInt   `json:"current_value"`
	Destroyed    bool           `json:"destroyed"`
}

type TokensInfo []*TokenInfo
