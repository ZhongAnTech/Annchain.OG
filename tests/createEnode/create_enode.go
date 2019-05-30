package main

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p/onode"
	"net"
)

func main() {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate ephemeral node key: %v", err))
	}
	data := crypto.FromECDSA(key)
	fmt.Println("nodekey", hex.EncodeToString(data))
	node := onode.NewV4(&key.PublicKey, net.ParseIP("192.168.1.1"), 8001, 8001)
	fmt.Println(node)
}
