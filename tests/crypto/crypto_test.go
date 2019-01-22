package crypto

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"testing"
	"time"
)

func TestRawTx_Tx(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	types.Signer = signer
	var num = 10000
	var txs types.Txs
	var rawtxs types.RawTxs
	for i:=0;i<num;i++{
		tx  := types.RandomTx()
		pub,_ ,_:= signer.RandomKeyPair()
		tx.PublicKey = pub.Bytes[:]
		rawtxs = append(rawtxs,tx.RawTx())
	}
	start:= time.Now()
	for i:=0;i<num;i++{
		txs = append(txs, rawtxs[i].Tx())
	}
	fmt.Println("used time ",time.Now().Sub(start))

}


func TestRawTx_encode(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	types.Signer = signer
	var num = 10000
	var txs types.Txs
	type bytes struct {
		types.RawData
	}
	var rawtxs []bytes
	for i:=0;i<num;i++{
		tx  := types.RandomTx()
		pub,_ ,_:= signer.RandomKeyPair()
		tx.PublicKey = pub.Bytes[:]
		data,_ := tx.MarshalMsg(nil)
		rawtxs = append(rawtxs,bytes{data})
	}
	start:= time.Now()
	for i:=0;i<num;i++{
		var tx types.Tx
		_,err := tx.UnmarshalMsg(rawtxs[i].RawData)
		if err!=nil {
			panic(err)
		}
		txs = append(txs, &tx)
	}
	fmt.Println("used time ",time.Now().Sub(start))

}