package partner

import "github.com/annchain/OG/account"

// provide Dkg term that will be changed every term switching.
type DkgTermProvider interface {
	CurrentDkgTerm() uint32
}

type AccountNonceProvider interface {
	GetNonce(account *account.Account) uint64
}
