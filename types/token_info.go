package types

//go:generate msgp

//msgp:tuple TokenInfo
type TokenInfo struct {
	PublicOffering
	Sender Address
}

type TokensInfo []*TokenInfo
