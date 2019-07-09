package tx_types

import "github.com/annchain/OG/common"

//go:generate msgp

//msgp:tuple TokenInfo
type TokenInfo struct {
	PublicOffering
	Sender common.Address
}

type TokensInfo []*TokenInfo
