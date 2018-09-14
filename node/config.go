package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/p2p/nat"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"github.com/annchain/OG/p2p/discv5"
)

const (
	datadirPrivateKey = "nodekey" // Path within the datadir to the node's private key
	defaultMaxPeers   = 50
)

func getNodePrivKey() *ecdsa.PrivateKey {
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
	p2pConfig.NodeName = viper.GetString("p2p.node_name")
	p2pConfig.NodeDatabase = viper.GetString("p2p.node_db")
	bootNodes := viper.GetString("p2p.bootstrap_nodes")
	p2pConfig.BootstrapNodesV5 = parserV5Nodes(bootNodes)
	//p2pConfig.NoDiscovery = true
	//p2pConfig.DiscoveryV5 = true
	//p2pConfig.BootstrapNodesV5: config.BootstrapNodes.nodes,
	//ListenAddr:       ":0",
	p2pConfig.NAT = nat.Any()

	return &p2p.Server{Config: p2pConfig}
}

func parserNodes(nodeString string) []*discover.Node {
	nodeList := strings.Split(nodeString, ";")
	var nodes []*discover.Node
	for _, url := range nodeList {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			log.Error(fmt.Sprintf("node URL %s: %v\n", url, err))
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
