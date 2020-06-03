package node

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og/account"
	"github.com/annchain/OG/plugin/community"
	"github.com/latifrons/soccerdash"
	"github.com/spf13/viper"
)

// Node is the basic entrypoint for all modules to start.
type Node2 struct {
	PrivateInfoProvider  account.PrivateInfoProvider
	PhysicalCommunicator community.LibP2pPhysicalCommunicator
	components           []Component
	networkId            uint32
}

func (n *Node2) InitDefault() {
	n.components = []Component{}
}

func (n *Node2) Setup() {
	n.PrivateInfoProvider = &core.LocalPrivateInfoProvider{}

	hostname := utilfuncs.getHostname()

	// load identity from config
	reporter := &soccerdash.Reporter{
		Id:            fmt.Sprintf(hostname),
		TargetAddress: viper.GetString("report.url"),
	}

	n.components = append(n.components, reporter)

	// Node must know who he is first
	privateInfo := n.PrivateInfoProvider.PrivateInfo()
	// Init peer to peer

	PhysicalCommunicator

	// network id is configured either in config.toml or env variable
	n.networkId = viper.GetUint32("p2p.network_id")
	if n.networkId == 0 {
		n.networkId = defaultNetworkId
	}

}
