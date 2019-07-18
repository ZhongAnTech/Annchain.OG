package cmd

import (
	"bufio"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types/tx_types"
	"github.com/spf13/cobra"
	"os"
	"strconv"
)

var (
	sigCmd = &cobra.Command{
		Use:   "sig",
		Short: "signature operator for txs.",
		Run:   sig,
	}
)

func sig(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("please enter your private key")
		return
	}
	privInput := args[0]
	privBytes, err := hexutil.Decode(privInput)
	if err != nil {
		fmt.Println("decode input private key to bytes error, check if it is a hex string: ", err)
		return
	}
	if len(privBytes) != 32 {
		fmt.Println(fmt.Sprintf("invalid private length, should be 32 bytes, get %d bytes.", len(privBytes)))
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("PLEASE ENTER nonce: ")
	nonceStr := ""
	if scanner.Scan() {
		nonceStr = scanner.Text()
	}
	nonce, err := strconv.Atoi(nonceStr)
	if err != nil {
		fmt.Println(fmt.Sprintf("nonce must be integer, err: %v", err))
		return
	}

	fmt.Println("PLEASE ENTER from: ")
	fromStr := ""
	if scanner.Scan() {
		fromStr = scanner.Text()
	}
	fromBytes, err := hexutil.Decode(fromStr)
	if err != nil {
		fmt.Println(fmt.Sprintf("from must be hex, err: %v", err))
		return
	}
	if len(fromBytes) != 20 {
		fmt.Println(fmt.Sprintf("invalid from length, must be 20 bytes, but get: %d", len(fromBytes)))
		return
	}
	from := common.BytesToAddress(fromBytes)

	fmt.Println("PLEASE ENTER value: ")
	valueStr := ""
	if scanner.Scan() {
		valueStr = scanner.Text()
	}
	valueInt, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Println(fmt.Sprintf("value must be integer, err: %v", err))
		return
	}
	value := math.NewBigInt(int64(valueInt))

	fmt.Println("PLEASE ENTER data: ")
	dataStr := ""
	if scanner.Scan() {
		dataStr = scanner.Text()
	}
	var data []byte
	if len(dataStr) > 0 {
		data, err = hexutil.Decode(dataStr)
		if err != nil {
			fmt.Println(fmt.Sprintf("data must be hex, err: %v", err))
			return
		}
	}

	tx := tx_types.Tx{}
	tx.AccountNonce = uint64(nonce)
	tx.From = &from
	tx.Value = value
	tx.Data = data

	sig := sign(privBytes, tx.SignatureTargets())
	if !sigVerify(privBytes, tx.SignatureTargets(), sig) {
		fmt.Println("signature failed, the signature cannot pass the verify.")
	}
	fmt.Println(fmt.Sprintf("\nsignature is: %x", sig.Bytes))
}

func sign(priv []byte, msg []byte) crypto.Signature {
	privateKey := crypto.PrivateKeyFromBytes(crypto.CryptoTypeSecp256k1, priv)
	signer := crypto.SignerSecp256k1{}

	return signer.Sign(privateKey, msg)
}

func sigVerify(priv []byte, msg []byte, sig crypto.Signature) bool {
	privateKey := crypto.PrivateKeyFromBytes(crypto.CryptoTypeSecp256k1, priv)

	signer := crypto.SignerSecp256k1{}
	publicKey := signer.PubKey(privateKey)

	return signer.Verify(publicKey, sig, msg)
}
