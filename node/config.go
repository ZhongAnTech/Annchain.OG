package node

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

func getNodePrivKey() *ecdsa.PrivateKey {
	return &ecdsa.PrivateKey{}
}

func NewP2PServer( privKey *ecdsa.PrivateKey) *p2p.Server {
	var p2pConfig p2p.Config
	p2pConfig.PrivateKey = privKey
	port := viper.GetString("port")
	p2pConfig.ListenAddr = ":" + port
	staticNodes := viper.GetString("static_nodes")
	p2pConfig.StaticNodes = parserNodes(staticNodes)
	trustNode := viper.GetString("trust_nodes")
	p2pConfig.TrustedNodes = parserNodes(trustNode)
	p2pConfig.Name = viper.GetString("node_name")
	p2pConfig.NodeDatabase = viper.GetString("node_db")
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
