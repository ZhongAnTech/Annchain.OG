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
		Use: "onode",
		Short: "onode helps you generate a random onode address",
		Run: genOnode,
	}
	output string
)

func onodeInit() {
	onodeCmd.Flags().StringVarP(&output, "output", "o", "", "")
}

func genOnode(cmd *cobra.Command, args []string) {
	var bootUrl = "47.100.222.11"

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate ephemeral node key: %v", err))
	}
	nodekey := hex.EncodeToString(crypto.FromECDSA(key))
	node := onode.NewV4(&key.PublicKey, net.ParseIP(bootUrl), 8001, 8001)

	fmt.Println("nodekey: ", nodekey)
	fmt.Println("ndoe: ", node.String())
}