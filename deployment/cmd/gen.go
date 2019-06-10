package cmd

import (
	"encoding/hex"
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
)

var configFileName = "config.toml"

var mainNetBootstrap = "onode://d2469187c351fad31b84f4afb2939cb19c03b7c9359f07447aea3a85664cd33d39afc0c531ad4a8e9ff5ed76b58216e19b0ba208b45d5017ca40c9bd351d29ee@47.100.222.11:8001"
var mainNetGenesisPk = "0x0104dbae4756ba4688b1d5637b97e07ce2502a55455f8494665ec9d73dbb3bf65ac0186f042ec77c69df44e2be60a6ab1715b33c852a13d37acc568562b712411ebf;0x01042e4d0369975c98a8880314994e9cbe8f45b1a374c75b98de776f5e9df57793249e7ad16c395d56cad540f660c9e18a04463f3888e4277dffbc317b31f7c61567;0x0104113f6d149d3b5ca450182a6af302aa4c59ae9678189608f4dc3f9139698600abac1b8490613bae74d389ce4266b9dae91caaef26d9b4e7a6ce867c8dfbfdec2d;0x0104062c0ee4be8f4965dd3553f9437b5f53954b391482abf05f7d89c9a1153f86e58b57b4497ca0030dd30658951aa695975c613367191cef16bc3f268f7122fe7d"

func genInit() {
	//genCmd.PersistentFlags().BoolVarP(&normal,"normal", "m", true, "normal node that connect to main network")
	genCmd.PersistentFlags().BoolVarP(&solo, "solo", "s", false, "solo node that use auto client to produce sequencer")
	genCmd.PersistentFlags().BoolVarP(&private, "private", "p", false, "private nodes that use your own boot-strap nodes")
	genCmd.PersistentFlags().IntVarP(&nodesNum, "node_num", "n", 4, "the number of nodes that will participate in consensus system")
}

func gen(cmd *cobra.Command, args []string) {
	readConfig()

	if private {
		nodekeyBoot, nodeBoot := genBootONode()
		viper.Set("bootstrap_nodes", nodeBoot)

		//generate consensus group keys
		privateSet := []string{}
		publicSet := []string{}
		for i := 0; i < nodesNum; i++ {
			priv, pub := genAccount()
			privateSet = append(privateSet, fmt.Sprintf("%x", priv))
			publicSet = append(publicSet, fmt.Sprintf("%x", pub))
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
		viper.Set("node_key", nodekeyBoot)
		err = mkDirIfNotExists(privateDir + "/node_boot")
		if err != nil {
			fmt.Println(fmt.Sprintf("check and make dir %s error: %v", privateDir+"/node_boot", err))
			return
		}
		viper.WriteConfigAs(privateDir + "/node_boot/" + configFileName)

		//init other nodes
		viper.Set("annsensus.campain", false)
		viper.Set("node_key", "")
		for i := 1; i < len(privateSet); i++ {
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
		viper.Set("dag.my_private_key", hex.EncodeToString(priv))

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
		viper.Set("dag.my_private_key", hex.EncodeToString(priv))

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

func genAccount() ([]byte, []byte) {
	signer := &crypto.SignerSecp256k1{}
	pub, priv := signer.RandomKeyPair()

	return priv.Bytes, pub.Bytes
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
