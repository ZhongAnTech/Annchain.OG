package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/spf13/cobra"
)

var (
	accountCmd = &cobra.Command{
		Use:   "account",
		Short: "account  operations for account",
	}

	accountGenCmd = &cobra.Command{
		Use:   "gen",
		Short: "account  operations for account",
		Run:   accountGen,
	}

	accountCalCmd = &cobra.Command{
		Use:   "cal",
		Short: "account  operations for account",
		Run:   accountCal,
	}
	priv_key string
)

func accountInit() {
	accountCmd.AddCommand(accountGenCmd, accountCalCmd)
	accountCalCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
}

func accountGen(cmd *cobra.Command, args []string) {
	signer := &crypto.SignerEd25519{}
	pub, priv, err := signer.RandomKeyPair()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%X\n", priv.Bytes[:])
	fmt.Printf("%X\n", pub.Bytes[:])
}

func accountCal(cmd *cobra.Command, args []string) {

	if priv_key == "" {
		fmt.Println("need private key ")
		return
	}

	data, err := hex.DecodeString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	priv := crypto.PrivateKey{
		Type:  crypto.CryptoTypeEd25519,
		Bytes: data,
	}
	signer := &crypto.SignerEd25519{}
	pub := signer.PubKey(priv)
	addr := signer.Address(pub)
	fmt.Printf("%X\n", addr.Bytes[:])
}
