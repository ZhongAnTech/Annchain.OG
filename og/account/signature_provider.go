package account

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
)

type SignatureProvider interface {
	Sign(data []byte) []byte
}

// AccountProvider provides public key and private key for signing consensus messages
type AccountProvider interface {
	Account() *account.Account
}

type AccountSignerSignatureProvider struct {
	Signer          crypto.ISigner
	AccountProvider AccountProvider
}

func NewAccountSignerSignatureProvider(signer crypto.ISigner, myAccountProvider AccountProvider) *AccountSignerSignatureProvider {
	return &AccountSignerSignatureProvider{
		Signer:          signer,
		AccountProvider: myAccountProvider,
	}
}

func (a AccountSignerSignatureProvider) Sign(data []byte) hexutil.Bytes {
	acc := a.AccountProvider.Account()
	if acc == nil {
		panic("account for signing cannot be nil")
	}
	return a.Signer.Sign(acc.PrivateKey, data).Bytes
}
