package cmd

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/poc/hotstuff_event"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"math/rand"
)

// runCmd represents the run command
var myCmd = &cobra.Command{
	Use:   "my",
	Short: "Generate my node info",
	Long:  `Generate my node info such as private key, public key and so on`,
	Run: func(cmd *cobra.Command, args []string) {
		mei := viper.GetInt64("mei")

		var r io.Reader

		if mei == -1 {
			r = crand.Reader
		} else {
			r = rand.New(rand.NewSource(mei))
		}

		priv, _, err := core.GenerateKeyPairWithReader(core.Secp256k1, 0, r)
		if err != nil {
			panic(err)
		}

		privm, err := core.MarshalPrivateKey(priv)
		if err != nil {
			panic(err)
		}
		privs := hex.EncodeToString(privm)

		opts := []libp2p.Option{
			libp2p.Identity(priv),
		}

		ctx := context.Background()

		node, err := libp2p.New(ctx, opts...)
		if err != nil {
			panic(err)
		}

		pi := &hotstuff_event.PrivateInfo{
			Type:       "secp256k1",
			PrivateKey: privs,
			Id:         node.ID().String(),
		}

		bytes, err := json.MarshalIndent(pi, "", " ")
		if err != nil {
			panic(err)
		}
		fmt.Println(viper.GetString("output"))
		err = ioutil.WriteFile(viper.GetString("output"), bytes, 0644)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(myCmd)
	myCmd.Flags().Int64P("mei", "i", -1, "my partner id starts from 0. If -1 is given, generate a random one")
	_ = viper.BindPFlag("mei", myCmd.Flags().Lookup("mei"))

	myCmd.Flags().StringP("output-file", "o", "id.key", "output file")
	_ = viper.BindPFlag("output", myCmd.Flags().Lookup("output-file"))

}
