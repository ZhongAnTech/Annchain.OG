package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/io"
	"github.com/annchain/OG/core"
	"github.com/spf13/viper"
)

const (
	PrivateServerDir = "private_server"
	PrivateDir       = "private"
	SoloDir          = "solo"
	MainNetDir       = "main_net"
	PrivateKeyFile   = "privkey"
)

type GenerateParams struct {
	Port                   int
	ConfigDir              string
	BootNodeIp             string
	EmbeddedBootstrap      bool
	NodesNum               int
	ConfigFileName         string
	IncreasePort           bool
	NetworkId              uint64
	ArchiveMode            bool
	ConfigTemplateFilename string
}

type Generator struct {
	GenerateParams
	viper *viper.Viper
}

func NewGenerator(params GenerateParams) *Generator {
	return &Generator{GenerateParams: params}
}

func (g *Generator) readConfig(configFileName string) error {
	file, err := os.Open(configFileName)
	if err != nil {
		return err
	}
	defer file.Close()
	v := viper.New()

	v.SetConfigType("toml")
	err = v.MergeConfig(file)
	if err != nil {
		return err
	}
	g.viper = v
	return nil
}

//PrivateChainConfig: used by sdk to autogenerate config
func (g *Generator) PrivateChainConfig() (err error) {
	err = g.readConfig(g.ConfigTemplateFilename)
	if err != nil {
		return err
	}
	nodeKeyBoot, nodeBoot := genBootONode(g.Port+1, g.BootNodeIp)
	if g.EmbeddedBootstrap {
		g.viper.Set("p2p.bootstrap_nodes", nodeBoot)
	}
	g.viper.Set("rpc.port", g.Port)
	g.viper.Set("p2p.port", g.Port+1)
	g.viper.Set("p2p.network_id", g.NetworkId)
	g.viper.Set("websocket.port", g.Port+2)
	g.viper.Set("profiling.port", g.Port+3)
	if g.ArchiveMode {
		g.viper.Set("mode", "archive")
	}

	//generate consensus group keys
	var privateSet []string
	var publicSet []string
	for i := 0; i < g.NodesNum; i++ {
		priv, pub := account.GenAccount()
		privateSet = append(privateSet, priv.String())
		publicSet = append(publicSet, pub.String())
	}
	genesisPk := strings.Join(publicSet, ";")
	g.viper.Set("annsensus.genesis_pk", genesisPk)
	g.viper.Set("annsensus.campaign", true)
	g.viper.Set("annsensus.partner_number", g.NodesNum)
	g.viper.Set("annsensus.threshold", 2*g.NodesNum/3+1)

	err = io.MkDirIfNotExists(g.ConfigDir)
	if err != nil {
		err = fmt.Errorf("check and make dir %s error: %v", g.ConfigDir, err)
		return
	}

	privateDirNode0 := path.Join(g.ConfigDir, "node_0")
	err = io.MkDirIfNotExists(privateDirNode0)
	if err != nil {
		err = fmt.Errorf("check and make dir %s error: %v", privateDirNode0, err)
		return
	}
	// init private key
	// g.viper.Set("dag.my_private_key", privateSet[0])
	account.SavePrivateKey(path.Join(privateDirNode0, PrivateKeyFile), privateSet[0])

	//init bootstrap
	g.viper.Set("p2p.node_key", nodeKeyBoot)
	g.viper.Set("p2p.bootstrap_node", true)
	g.viper.Set("leveldb.path", "rw/datadir_0")
	g.viper.Set("annsensus.consensus_path", "consensus0.json")
	err = g.viper.WriteConfigAs(path.Join(privateDirNode0, g.ConfigFileName))
	if err != nil {
		err = fmt.Errorf("error on dump config %v", err)
		return
	}

	// copy genesis
	if io.FileExists("genesis.json") {
		io.CopyFile("genesis.json", path.Join(privateDirNode0, "genesis.json"))
	} else {
		ioutil.WriteFile(path.Join(privateDirNode0, "genesis.json"), Defaultgenesis(), 0644)
	}

	//init other nodes
	g.viper.Set("annsensus.campaign", true)
	g.viper.Set("p2p.bootstrap_node", false)
	portGap := 0
	if g.IncreasePort {
		portGap = 10
	}
	for i := 1; i < len(privateSet); i++ {
		g.viper.Set("rpc.port", g.Port+portGap*i)
		g.viper.Set("p2p.port", g.Port+portGap*i+1)
		g.viper.Set("websocket.port", g.Port+portGap*i+2)
		g.viper.Set("profiling.port", g.Port+portGap*i+3)
		g.viper.Set("leveldb.path", fmt.Sprintf("rw/datadir_%d", i))
		g.viper.Set("annsensus.consensus_path", fmt.Sprintf("consensus%d.json", i))
		nodekey, _ := genBootONode(g.Port+portGap*i+1, "127.0.0.1")
		g.viper.Set("p2p.node_key", nodekey)

		configDir := path.Join(g.ConfigDir, fmt.Sprintf("/node_%d", i))
		err = io.MkDirIfNotExists(configDir)
		if err != nil {
			err = fmt.Errorf("check and make dir %s error: %v", configDir, err)
			return
		}
		account.SavePrivateKey(path.Join(configDir, PrivateKeyFile), privateSet[i])
		err = g.viper.WriteConfigAs(path.Join(configDir, g.ConfigFileName))
		if err != nil {
			err = fmt.Errorf("error on dump config %v", err)
			return
		}
		if io.FileExists("genesis.json") {
			io.CopyFile("genesis.json", path.Join(configDir, "genesis.json"))
		} else {
			ioutil.WriteFile(path.Join(configDir, "genesis.json"), Defaultgenesis(), 0644)
		}

	}
	return nil
}

func Defaultgenesis() []byte {
	addrs := []string{"0x49fdaab0af739e16c9e1c9bf1715a6503edf4cab",
		"0xdb8bbcb4ddd236924f4fbe0018274f3c5041a8c3",
		"0x3928847b3e41ddc942c7c028463ba813f761f688",
		"0x682b988e1952c248bf8c6c3da95ffce319e316b5",
		"0x9ae4e9621e71044bb02bf5d800e915cca65901ae",
	}
	accounts := core.GenesisAccounts{}
	for _, v := range addrs {
		accounts.Accounts = append(accounts.Accounts, core.Account{Address: v, Balance: 1000000})
	}
	data, _ := json.Marshal(&accounts)
	return data
}
