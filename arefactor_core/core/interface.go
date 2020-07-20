package core

import (
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/OG/common/math"
)

type LedgerEngine interface {
	GetNonce(ogTypes.Address) uint64
	SetNonce(ogTypes.Address, uint64)
	IssueToken(issuer ogTypes.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt) (int32, error)
	ReIssueToken(tokenID int32, amount *math.BigInt) error
	DestroyToken(tokenID int32) error

	SubTokenBalance(ogTypes.Address, int32, *math.BigInt)
	AddTokenBalance(ogTypes.Address, int32, *math.BigInt)
}

type TxProcessor interface {
	ProcessTransaction(engine LedgerEngine, tx types.Txi) (*Receipt, error)
}
