package account

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/encryption"
)

func SavePrivateKey(path string, content string) {
	vault := encryption.NewVault([]byte(content))
	if err := vault.Dump(path, ""); err != nil {
		fmt.Println(fmt.Sprintf("error on saving privkey to %s: %v", path, err))
		panic(err)
	}
}

func GenAccount() (crypto.PrivateKey, crypto.PublicKey) {
	signer := &crypto.SignerSecp256k1{}
	pub, priv := signer.RandomKeyPair()

	return priv, pub
}

func RandomAccount() *Account {
	sk, pk := GenAccount()
	signer := crypto.NewSigner(sk.Type)

	return &Account{
		Id:          0,
		PrivateKey:  sk,
		PublicKey:   pk,
		Address:     signer.Address(pk),
		InitBalance: 0,
	}
}
