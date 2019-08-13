// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
	accountCmd.PersistentFlags().StringVarP(&algorithm, "algorithm", "a", "secp256k1", "algorithm e (ed25519) ; algorithm s (secp256k1")
	accountCalCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
}

func accountGen(cmd *cobra.Command, args []string) {
	var signer crypto.ISigner
	if algorithm == "secp256k1" || algorithm == "s" {
		signer = &crypto.SignerSecp256k1{}
	} else if algorithm == "ed25519" || algorithm == "e" {
		signer = &crypto.SignerEd25519{}
	} else {
		fmt.Println("unknown crypto algorithm", algorithm)
		return
	}
	pub, priv := signer.RandomKeyPair()
	fmt.Println(priv.String())
	fmt.Println(pub.String())
	fmt.Println(signer.Address(pub).Hex())
}

func accountCal(cmd *cobra.Command, args []string) {

	if priv_key == "" {
		fmt.Println("need private key")
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
	fmt.Println(pub.String())
	fmt.Println(addr.String())
}
