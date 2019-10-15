package tx_client

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/og/verifier"
	"github.com/annchain/OG/rpc"
)

type RequstGenerator struct {
	privKey   crypto.PrivateKey
	publicKey *crypto.PublicKey
	address   common.Address
	Nodebug   bool
}

func (r *RequstGenerator) Address() common.Address {
	return r.address
}

func NewRequestGenerator(priv crypto.PrivateKey) *RequstGenerator {
	return &RequstGenerator{
		privKey:   priv,
		publicKey: priv.PublicKey(),
		address:   priv.PublicKey().Address(),
	}
}

func (r *RequstGenerator) TokenPublishing(nonce uint64, enableSPO bool, tokenName string, value *math.BigInt) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	from := r.address
	if !r.Nodebug {
		fmt.Println(from.String())
	}
	tx := protocol_message.ActionTx{
		TxBase: protocol_message.TxBase{
			Type:         protocol_message.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    r.publicKey.Bytes[:],
		},
		Action: protocol_message.ActionTxActionIPO,
		From:   &from,
		ActionData: &protocol_message.PublicOffering{
			Value:     value,
			EnableSPO: enableSPO,
			TokenName: tokenName,
		},
	}
	tx.Signature = crypto.Signer.Sign(r.privKey, tx.SignatureTargets()).Bytes[:]
	v := verifier.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     nonce,
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    r.publicKey.String(),
		Action:    protocol_message.ActionTxActionIPO,
		EnableSPO: enableSPO,
		TokenName: tokenName,
	}

	return request
}

func (r *RequstGenerator) TokenDestroy(tokenId int32, nonce uint64) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	from := r.address
	//fmt.Println(pub.String(), priv.String(), from.String())
	fmt.Println(from.String())
	value := math.NewBigInt(0)

	tx := protocol_message.ActionTx{
		TxBase: protocol_message.TxBase{
			Type:         protocol_message.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    r.publicKey.Bytes[:],
		},
		Action: protocol_message.ActionTxActionDestroy,
		From:   &from,
		ActionData: &protocol_message.PublicOffering{
			Value: value,
			//EnableSPO:  enableSPO,
			//TokenName: "test_token",
			TokenId: tokenId,
		},
	}
	tx.Signature = crypto.Signer.Sign(r.privKey, tx.SignatureTargets()).Bytes[:]
	v := verifier.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     nonce,
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    r.publicKey.String(),
		Action:    protocol_message.ActionTxActionDestroy,
		//EnableSPO: enableSPO,
		//TokenName: "test_token",
		TokenId: tokenId,
	}

	return request
}

func (r *RequstGenerator) SecondPublicOffering(tokenId int32, nonce uint64, value *math.BigInt) rpc.NewPublicOfferingRequest {
	//pub, priv := crypto.Signer.RandomKeyPair()
	from := r.address
	if !r.Nodebug {
		fmt.Println(from.String())
	}
	tx := protocol_message.ActionTx{
		TxBase: protocol_message.TxBase{
			Type:         protocol_message.TxBaseAction,
			AccountNonce: uint64(nonce),
			PublicKey:    r.publicKey.Bytes[:],
		},
		Action: protocol_message.ActionTxActionSPO,
		From:   &from,
		ActionData: &protocol_message.PublicOffering{
			Value: value,
			//EnableSPO: true,
			//TokenName: "test_token",
			TokenId: tokenId,
		},
	}
	tx.Signature = crypto.Signer.Sign(r.privKey, tx.SignatureTargets()).Bytes[:]
	v := verifier.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	target := tx.SignatureTargets()
	fmt.Println(hexutil.Encode(target))
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewPublicOfferingRequest{
		Nonce:     nonce,
		From:      tx.From.Hex(),
		Value:     value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    r.publicKey.String(),
		Action:    protocol_message.ActionTxActionSPO,
		//EnableSPO: true,
		//TokenName: "test_token",
		TokenId: tokenId,
	}

	return request
}

func (r *RequstGenerator) NormalTx(tokenId int32, nonce uint64, to common.Address, value *math.BigInt) rpc.NewTxRequest {
	from := r.address
	if !r.Nodebug {
		fmt.Println(from.String(), to.String())
	}
	tx := protocol_message.Tx{
		TxBase: protocol_message.TxBase{
			Type:         protocol_message.TxBaseTypeNormal,
			PublicKey:    r.publicKey.Bytes[:],
			AccountNonce: uint64(nonce),
		},
		From:    &from,
		TokenId: tokenId,
		Value:   value,
		To:      to,
	}
	tx.Signature = crypto.Signer.Sign(r.privKey, tx.SignatureTargets()).Bytes[:]
	v := verifier.TxFormatVerifier{}
	ok := v.VerifySignature(&tx)
	if !ok {
		target := tx.SignatureTargets()
		fmt.Println(hexutil.Encode(target))
		panic("not ok")
	}
	request := rpc.NewTxRequest{
		Nonce:     fmt.Sprintf("%d", nonce),
		From:      tx.From.Hex(),
		To:        to.String(),
		Value:     tx.Value.String(),
		Signature: tx.Signature.String(),
		Pubkey:    r.publicKey.String(),
		TokenId:   tokenId,
	}
	return request
}
