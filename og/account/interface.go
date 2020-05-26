package account

import "github.com/annchain/OG/account"

type AccountNonceProvider interface {
	GetNonce(account *account.Account) uint64
}

// AccountProvider provides public key and private key for signing consensus messages
type AccountProvider interface {
	Account() *account.Account
}

type SignatureProvider interface {
	Sign(data []byte) (publicKey []byte, signature []byte)
}

type PrivateInfoProvider interface {
	PrivateInfo() *account.PrivateInfo
}