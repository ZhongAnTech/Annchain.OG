package cmd

import (
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
	priv_key  string
	algorithm string
)

func accountInit() {
	accountCmd.AddCommand(accountGenCmd, accountCalCmd)
	accountCmd.PersistentFlags().StringVarP(&algorithm, "algorithm", "a", "ed25519", "algorithm e (ed25519) ; algorithm s (secp256k1")
	accountCalCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
}

func accountGen(cmd *cobra.Command, args []string) {
	if algorithm == "secp256k1" || algorithm == "s" {
		signer := &crypto.SignerSecp256k1{}
		pub, priv, err := signer.RandomKeyPair()
		if err != nil {
			panic(err)
		}
		fmt.Printf(priv.PrivateKeyToString())
		fmt.Printf(pub.PublicKeyToString())
	} else {
		signer := &crypto.SignerEd25519{}
		pub, priv, err := signer.RandomKeyPair()
		if err != nil {
			panic(err)
		}
		fmt.Printf(priv.PrivateKeyToString())
		fmt.Printf(pub.PublicKeyToString())
	}

}

func accountCal(cmd *cobra.Command, args []string) {

	if priv_key == "" {
		fmt.Println("need private key ")
		return
	}
	privKey, err := crypto.PrivateKeyFromString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	signer := crypto.NewSigner(privKey.Type)
	pub := signer.PubKey(privKey)
	addr := signer.Address(pub)
	fmt.Println(pub.PublicKeyToString())
	fmt.Println(addr.String())
}
