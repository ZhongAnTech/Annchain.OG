package account

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
)

type SampleAccount struct {
	Id         int
	PrivateKey crypto.PrivateKey
	PublicKey  crypto.PublicKey
	Address    types.Address
	Nonce      uint64
}

func NewAccount(privateKeyHex string) SampleAccount {
	signer := &crypto.SignerSecp256k1{}

	s := SampleAccount{}
	pv, err := crypto.PrivateKeyFromString(privateKeyHex)
	if err != nil {
		panic(err)
	}
	s.PrivateKey = pv
	s.PublicKey = signer.PubKey(pv)
	s.Address = signer.Address(s.PublicKey)
	return s

}
