package types

//go:generate msgp

type TokenInfo struct {
	PublicOffering
	Sender Address
}

type TokensInfo []*TokenInfo
