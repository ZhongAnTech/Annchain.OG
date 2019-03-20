package genesis_account

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"io/ioutil"
	"testing"
)

type PublicAccount struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
}

type Account struct {
	PublicAccount
	PublicKey  string
	PrivateKey string
}

type genesis struct {
	Accounts    []PublicAccount `json:"accounts"`
}

type secretGenesis struct {
	Accounts    []Account `json:"accounts"`
}

func TestAccount(t *testing.T) {
	signer := crypto.SignerSecp256k1{}
	var se secretGenesis
	var ge genesis
	for i := 0; i < 7; i++ {
		pub, priv, err := signer.RandomKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		publicAccount := PublicAccount{
			Address: pub.Address().String(),
			Balance: 1000000,
		}
		account := Account{
			PublicAccount: publicAccount,
			PrivateKey:    priv.String(),
			PublicKey:     pub.String(),
		}
		se.Accounts = append(se.Accounts, account)
		ge.Accounts = append(ge.Accounts, publicAccount)
	}
	_, priv, err := signer.RandomKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(ge, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("genesis.json", data, 0644)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.MarshalIndent(se, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("secret.json", data, 0644)
	if err != nil {
		t.Fatal(err)
	}

	var se2 secretGenesis
	err = json.Unmarshal(data, &se2)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := json.MarshalIndent(se2, "", "\t")
	fmt.Println(string(d))

}

func TestNewAccount(t *testing.T) {
	signer := crypto.SignerSecp256k1{}
	pub, priv, err := signer.RandomKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(pub, priv)
}
