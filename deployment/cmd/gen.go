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
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var (
	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "generate config.toml files for deployment",
		Run:   gen,
	}
	//normal bool
	solo     bool
	private  bool
	nodesNum int
)

var (
	privateDir = "private"
	soloDir    = "solo"
	mainNetDir = "main_net"
	port int  = 8000
)

var configFileName = "config.toml"

var mainNetBootstrap = "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001"
var mainNetGenesisPk = "0x0104dbae4756ba4688b1d5637b97e07ce2502a55455f8494665ec9d73dbb3bf65ac0186f042ec77c69df44e2be60a6ab1715b33c852a13d37acc568562b712411ebf;0x01042e4d0369975c98a8880314994e9cbe8f45b1a374c75b98de776f5e9df57793249e7ad16c395d56cad540f660c9e18a04463f3888e4277dffbc317b31f7c61567;0x0104113f6d149d3b5ca450182a6af302aa4c59ae9678189608f4dc3f9139698600abac1b8490613bae74d389ce4266b9dae91caaef26d9b4e7a6ce867c8dfbfdec2d;0x0104062c0ee4be8f4965dd3553f9437b5f53954b391482abf05f7d89c9a1153f86e58b57b4497ca0030dd30658951aa695975c613367191cef16bc3f268f7122fe7d"

func genInit() {
	//genCmd.PersistentFlags().BoolVarP(&normal,"normal", "m", true, "normal node that connect to main network")
	genCmd.PersistentFlags().BoolVarP(&solo, "solo", "s", false, "solo node that use auto client to produce sequencer")
	genCmd.PersistentFlags().BoolVarP(&private, "private", "p", false, "private nodes that use your own boot-strap nodes")
	genCmd.PersistentFlags().IntVarP(&nodesNum, "node_num", "n", 4, "the number of nodes that will participate in consensus system")
	genCmd.PersistentFlags().IntVarP(&port,"port","t",8000,"the port of private network")
}

func gen(cmd *cobra.Command, args []string) {
	readConfig()

	if private {
		nodekeyBoot, nodeBoot := genBootONode(port+1)
		viper.Set("p2p.bootstrap_nodes", nodeBoot)
		viper.Set("rpc.port",port)
		viper.Set("p2p.port",port+1)
		viper.Set("websocket.port",port+2)
		viper.Set("profiling.port",port+3)

		//generate consensus group keys
		privateSet := []string{}
		publicSet := []string{}
		for i := 0; i < nodesNum; i++ {
			priv, pub := genAccount()
			privateSet = append(privateSet, priv.String())
			publicSet = append(publicSet,  pub.String())
		}
		genesisPk := strings.Join(publicSet, ";")
		viper.Set("annsensus.genesis_pk", genesisPk)
		viper.Set("annsensus.campain", true)

		err := mkDirIfNotExists(privateDir)
		if err != nil {
			fmt.Println(fmt.Sprintf("check and make dir %s error: %v", privateDir, err))
			return
		}

		//init bootstrap
		viper.Set("dag.my_private_key", privateSet[0])
		viper.Set("p2p.node_key", nodekeyBoot)
		viper.Set("p2p.bootstrap_node",true)
		err = mkDirIfNotExists(privateDir + "/node_0")
		if err != nil {
			fmt.Println(fmt.Sprintf("check and make dir %s error: %v", privateDir+"/node_0", err))
			return
		}
		viper.Set("leveldb.path","rw/datadir_0")
		viper.Set("annsensus.consensus_path","consensus0.json")
		viper.WriteConfigAs(privateDir + "/node_0/" + configFileName)

		//init other nodes
		viper.Set("annsensus.campain", false)
		viper.Set("p2p.bootstrap_node",false)
		for i := 1; i < len(privateSet); i++ {
			viper.Set("rpc.port",port+10*i)
			viper.Set("p2p.port",port+10*i+1)
			viper.Set("websocket.port",port+10*i+2)
			viper.Set("profiling.port",port+10*i+3)
			viper.Set("leveldb.path",fmt.Sprintf("rw/datadir_%d",i))
			viper.Set("annsensus.consensus_path",fmt.Sprintf("consensus%d.json",i))
			nodekey ,_:= genBootONode(port+10*i+1)
			viper.Set("p2p.node_key", nodekey)
			fmt.Println("private key: ", i)
			viper.Set("dag.my_private_key", privateSet[i])

			configDir := privateDir + "/node_" + fmt.Sprintf("%d", i)
			err = mkDirIfNotExists(configDir)
			if err != nil {
				fmt.Println(fmt.Sprintf("check and make dir %s error: %v", configDir, err))
				return
			}
			viper.WriteConfigAs(configDir + "/" + configFileName)
		}

	} else if solo {
		viper.Set("annsensus.disable", true)
		viper.Set("auto_client.sequencer.enabled", true)

		priv, _ := genAccount()
		viper.Set("dag.my_private_key", priv.String())
		viper.Set("rpc.port",port)
		viper.Set("p2p.port",port+1)
		viper.Set("websocket.port",port+2)
		viper.Set("profiling.port",port+3)
		err := mkDirIfNotExists(soloDir)
		if err != nil {
			fmt.Println(fmt.Sprintf("check and make dir %s error: %v", soloDir, err))
			return
		}

		viper.WriteConfigAs(soloDir + "/" + configFileName)

	} else {
		viper.Set("bootstrap_nodes", mainNetBootstrap)
		viper.Set("annsensus.genesis_pk", mainNetGenesisPk)

		priv, _ := genAccount()
		viper.Set("rpc.port",port)
		viper.Set("p2p.port",port+1)
		viper.Set("websocket.port",port+2)
		viper.Set("profiling.port",port+3)
		viper.Set("dag.my_private_key", priv.String())

		err := mkDirIfNotExists(mainNetDir)
		if err != nil {
			fmt.Println(fmt.Sprintf("check and make dir %s error: %v", mainNetDir, err))
			return
		}

		viper.WriteConfigAs(mainNetDir + "/" + configFileName)
	}

}

func readConfig() {
	file, err := os.Open(configFileName)
	panicIfError(err, "open file error")
	defer file.Close()

	viper.SetConfigType("toml")
	err = viper.MergeConfig(file)

	panicIfError(err, "merge viper config err")
}

func genAccount() (crypto.PrivateKey,crypto.PublicKey) {
	signer := &crypto.SignerSecp256k1{}
	pub, priv := signer.RandomKeyPair()

	return priv, pub
}

func panicIfError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func mkDirIfNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(path, os.ModePerm)
}
