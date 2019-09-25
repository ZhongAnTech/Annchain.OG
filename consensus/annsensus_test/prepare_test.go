package annsensus_test

import (
	"fmt"
	"github.com/annchain/OG/account"
)

func sampleAccounts(count int) []*account.Account {
	var accounts []*account.Account
	for i := 0; i < count; i++ {
		acc := account.NewAccount(fmt.Sprintf("0x0170E6B713CD32904D07A55B3AF5784E0B23EB38589EBF975F0AB89E6F8D786F%02X", i))
		fmt.Println(fmt.Sprintf("account address: %s, pubkey: %s, privkey: %s", acc.Address.String(), acc.PublicKey.String(), acc.PrivateKey.String()))
		accounts = append(accounts, acc)
	}
	return accounts
}