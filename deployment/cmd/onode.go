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
		Run: genONode,
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