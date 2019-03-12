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
package node

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discv5"
	"github.com/annchain/OG/p2p/nat"
	"github.com/annchain/OG/p2p/onode"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const (
	datadirPrivateKey = "nodekey" // Path within the datadir to the node's private key
	defaultMaxPeers   = 50
	defaultNetworkId  = 1
)

func getNodePrivKey() *ecdsa.PrivateKey {
	nodeKey := viper.GetString("p2p.node_key")
	if nodeKey != "" {
		keyByte, err := hex.DecodeString(nodeKey)
		if err != nil {
			panic(fmt.Sprintf("get nodekey error %v ", err))
		}
		key, err := crypto.ToECDSA(keyByte)
		if err != nil {
			panic(fmt.Sprintf("get nodekey error %v ", err))
		}
		return key
	}
	datadir := viper.GetString("datadir")
	// Use any specifically configured key.

	// Generate ephemeral key if no datadir is being used.
	if datadir == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(fmt.Sprintf("failed to generate ephemeral node key: %v", err))
		}
		return key
	}

	keyfile := resolvePath(datadirPrivateKey)
	if key, err := crypto.LoadECDSA(keyfile); err == nil {
		return key
	}
	// No persistent key found, generate and store a new one.
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("failed to generate node key: %v", err))
	}
	if err := os.MkdirAll(datadir, 0700); err != nil {
		log.Error(fmt.Sprintf("failed to persist node key: %v", err))
		return key
	}
	keyfile = filepath.Join(datadir, datadirPrivateKey)
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		log.Error(fmt.Sprintf("failed to persist node key: %v", err))
	}
	return key
}

// ResolvePath resolves path in the instance directory.
func resolvePath(path string) string {
	datadir := viper.GetString("datadir")
	if filepath.IsAbs(path) {
		return path
	}
	oldpath := filepath.Join(datadir, path)
	if filepath.IsAbs(oldpath) {
		return oldpath
	}
	return oldpath

}

func NewP2PServer(privKey *ecdsa.PrivateKey) *p2p.Server {
	var p2pConfig p2p.Config
	p2pConfig.PrivateKey = privKey
	port := viper.GetString("p2p.port")
	p2pConfig.ListenAddr = ":" + port
	maxPeers := viper.GetInt("p2p.max_peers")
	if maxPeers <= 0 {
		maxPeers = defaultMaxPeers
	}
	p2pConfig.MaxPeers = maxPeers
	staticNodes := viper.GetString("p2p.static_nodes")
	p2pConfig.StaticNodes = parserNodes(staticNodes)
	trustNode := viper.GetString("p2p.trust_nodes")
	p2pConfig.TrustedNodes = parserNodes(trustNode)
	nodeName := viper.GetString("p2p.node_name")
	if nodeName == "" {
		nodeName = "og"
	}
	p2pConfig.NodeName = nodeName
	p2pConfig.NodeDatabase = viper.GetString("p2p.node_db")
	bootNodes := viper.GetString("p2p.bootstrap_nodes")
	bootNodesV5 := viper.GetString("p2p.bootstrap_nodes_v5")
	p2pConfig.BootstrapNodes = parserNodes(bootNodes)
	p2pConfig.BootstrapNodesV5 = parserV5Nodes(bootNodesV5)
	//p2pConfig.NoDiscovery = true
	//p2pConfig.DiscoveryV5 = true
	//p2pConfig.BootstrapNodesV5: config.BootstrapNodes.nodes,
	p2pConfig.NAT = nat.Any()
	p2pConfig.NoEncryption = viper.GetBool("p2p.no_encryption")

	return &p2p.Server{Config: p2pConfig}
}

func parserNodes(nodeString string) []*onode.Node {
	nodeList := strings.Split(nodeString, ";")
	var nodes []*onode.Node
	for _, url := range nodeList {
		if url == "" {
			continue
		}
		node, err := onode.ParseV4(url)
		if err != nil {
			log.Error(fmt.Sprintf("Node URL %s: %v\n", url, err))
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func parserV5Nodes(nodeString string) []*discv5.Node {
	nodeList := strings.Split(nodeString, ";")
	var nodes []*discv5.Node
	for _, url := range nodeList {
		if url == "" {
			continue
		}
		node, err := discv5.ParseNode(url)
		if err != nil {
			log.Error(fmt.Sprintf("node URL %s: %v\n", url, err))
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}


func parserGenesisAccounts (signer crypto.Signer, pubkeys string)  []crypto.PublicKey {
    pubkeyList  := strings.Split(pubkeys, ";")
    var account []crypto.PublicKey
    for _, pubKeyStr:= range pubkeyList {
		pubKey,err := crypto.PublicKeyFromString(pubKeyStr)
		if err!=nil {
			panic(err)
		}
		account = append(account,pubKey)
	}
   return  account
}

