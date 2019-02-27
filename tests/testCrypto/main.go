package main

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
)

func main() {
	//pHash := common.FromHex("0x3f2d2085f7ad5243118fa033675ee867c179582606b4f7e05b04749ca019d254")
	signer := crypto.NewSigner(crypto.CryptoTypeEd25519)
	privKey, err := crypto.PrivateKeyFromString(
		"0x009d9d0fe5e9ef0bb3bb4934db878688500fd0fd8e026c1ff1249b7e268c8a363aa7d45d13a5accb299dc7fe0f3b5fb0e9526b67008f7ead02c51c7b1f5a1d7b00")
	if err != nil {
		panic(err)
	}
	pubKey := signer.PubKey(privKey)
	tx := newUnsignedSequencer(signer.Address(pubKey), 1, nil, 0)
	// do sign work
	data := tx.SignatureTargets()
	signature := signer.Sign(privKey, data)
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = signer.PubKey(privKey).Bytes
	fmt.Println("dump ", tx.Dump())
	fmt.Println("data ", hexutil.Encode(data))
	fmt.Println("sig ", hexutil.Encode(signature.Bytes))
	//data2 :=tx.SignatureTargets()
	ok := signer.Verify(
		//crypto.PublicKey{Type: signer.GetCryptoType(), Bytes: tx.GetBase().PublicKey},
		pubKey,
		//crypto.Signature{Type: signer.GetCryptoType(), Bytes: tx.GetBase().Signature},
		signature,
		data)
	if !ok {
		panic(fmt.Sprintf("fail  %v", ok))
	}

}

func newUnsignedSequencer(issuer types.Address, id uint64, contractHashOrder types.Hashes, accountNonce uint64) types.Txi {
	tx := types.Sequencer{
		Issuer:            issuer,
		Id:                id,
		ContractHashOrder: contractHashOrder,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
		},
	}
	return &tx
}
