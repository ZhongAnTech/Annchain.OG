package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"path/filepath"
	"os"
	"github.com/annchain/OG/common/crypto"
)


const (
	datadirPrivateKey      = "nodekey"            // Path within the datadir to the node's private key
	defaultMaxPeers   = 50
)


func getNodePrivKey() *ecdsa.PrivateKey {
	datadir := viper.GetString("datadir")
	// Use any specifically configured key.

	// Generate ephemeral key if no datadir is being used.
	if datadir == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			panic(fmt.Sprintf("Failed to generate ephemeral node key: %v", err))
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
		panic(fmt.Sprintf("Failed to generate node key: %v", err))
	}
	if err := os.MkdirAll(datadir, 0700); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
		return key
	}
	keyfile = filepath.Join(datadir, datadirPrivateKey)
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		log.Error(fmt.Sprintf("Failed to persist node key: %v", err))
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
	return  oldpath

}



func NewP2PServer( privKey *ecdsa.PrivateKey) *p2p.Server {
	var p2pConfig p2p.Config
	p2pConfig.PrivateKey = privKey
	port := viper.GetString("p2p.port")
	p2pConfig.ListenAddr = ":" + port
	maxPeers :=  viper.GetInt("p2p.max_peers")
	if maxPeers <= 0 {
		maxPeers = 50
	}
	p2pConfig.MaxPeers = defaultMaxPeers
	staticNodes := viper.GetString("p2p.static_nodes")
	p2pConfig.StaticNodes = parserNodes(staticNodes)
	trustNode := viper.GetString("p2p.trust_nodes")
	p2pConfig.TrustedNodes = parserNodes(trustNode)
	p2pConfig.NodeName = viper.GetString("p2p.node_name")
	p2pConfig.NodeDatabase = viper.GetString("p2p.node_db")
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
			log.Error(fmt.Sprintf("Node URL %s: %v\n", url, err))
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}
