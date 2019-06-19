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
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p/onode"
	"github.com/spf13/cobra"
	"net"
)

var (
	onodeCmd = &cobra.Command{
		Use:   "onode",
		Short: "onode helps you generate a random onode address",
		Run:   genONode,
	}
	output string
)

func onodeInit() {
	onodeCmd.Flags().StringVarP(&output, "output", "o", "", "")
}

func genONode(cmd *cobra.Command, args []string) {
	nodekey, node := genBootONode()

	fmt.Println("nodekey: ", nodekey)
	fmt.Println("ndoe: ", node)
}

func genBootONode() (nodekey string, node string) {

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate ephemeral node key: %v", err))
	}
	nodekey = hex.EncodeToString(crypto.FromECDSA(key))
	node = onode.NewV4(&key.PublicKey, net.ParseIP(bootUrl), 8001, 8001).String()

	return
}
