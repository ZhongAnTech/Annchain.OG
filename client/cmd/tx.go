package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/spf13/cobra"
)

var (
	txCmd = &cobra.Command{
		Use:   "tx ",
		Short: "send new transaction",
		Run:   newTx,
	}

	payload string
	to      string
	nonce   uint64
	value   int64
)

func txInit() {
	txCmd.PersistentFlags().StringVarP(&payload, "payload", "p", "", "payload value")
	txCmd.PersistentFlags().StringVarP(&to, "to", "t", "", "to 0x***")
	txCmd.PersistentFlags().StringVarP(&priv_key, "priv_key", "k", "", "priv_key ***")
	txCmd.PersistentFlags().Int64VarP(&value, "value", "v", 0, "value 1")
	txCmd.PersistentFlags().Uint64VarP(&nonce, "nonce", "n", 0, "nonce 1")
	txCmd.PersistentFlags().StringVarP(&algorithm, "algorithm", "a", "ed25519", "algorithm e (ed25519) ; algorithm s (secp256k1")
}

func newTx(cmd *cobra.Command, args []string) {
	if to == "" || value < 1 || priv_key == "" || nonce < 0 {
		cmd.HelpFunc()
	}
	toAddr := types.HexToAddress(to)
	// fromAddr := types.HexToAddress(from)
	data, err := hex.DecodeString(priv_key)
	if err != nil {
		fmt.Println(err)
		return
	}
	key := crypto.PrivateKey{
		Bytes: data,
	}
	if algorithm =="secp256k1" || algorithm =="s" {
		key.Type = crypto.CryptoTypeSecp256k1
	}else {
		key.Type = crypto.CryptoTypeSecp256k1
	}

	//todo smart contracts
	//data := common.Hex2Bytes(payload)
	// do sign work
	var signer crypto.Signer

	if algorithm =="secp256k1" || algorithm =="s" {
		signer = &crypto.SignerSecp256k1{}
	}else {
		signer = &crypto.SignerEd25519{}
	}
	pub := signer.PubKey(key)
	from := signer.Address(pub)
	tx := types.Tx{
		Value: math.NewBigInt(value),
		To:    toAddr,
		From:  from,
		TxBase: types.TxBase{
			AccountNonce: nonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	signature := signer.Sign(key, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = signer.PubKey(key).Bytes
	req := httplib.Post(Host + "/new_transaction")
	_, err = req.JSONBody(&tx)
	if err != nil {
		panic(fmt.Errorf("encode tx errror %v", err))
	}
	str, err := req.String()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(str)
}
