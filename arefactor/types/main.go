package types

import (
	"fmt"
	common2 "github.com/annchain/OG/arefactor/common"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"math/big"
)

func main() {

	hash := common.RandomHash()
	parentHash := common.RandomHash()
	pubKey := PublicKey([]byte("lalalala"))
	sig := hexutil.Bytes([]byte("sigsigsig23456789098765432"))
	txBase := TxBase{
		Type:         TxBaseTypeNormal,
		Hash:         hash,
		ParentsHash:  common.Hashes{parentHash},
		AccountNonce: 1,
		Height:       1,
		PublicKey:    pubKey,
		Signature:    sig,
		MineNonce:    0,
		Weight:       0,
		inValid:      false,
		Version:      0,
		verified:     0,
	}
	from := common.RandomAddress()
	to := common.RandomAddress()
	value := math.NewBigInt(1234)
	tx := Tx{
		TxBase:  txBase,
		From:    &from,
		To:      to,
		Value:   value,
		TokenId: 0,
		Data:    nil,
	}

	//testTimes := 100000
	//
	//startTime := time.Now()
	//for i:=0; i<testTimes; i++ {
	//	msgpTest(tx)
	//}
	//timeGap := time.Since(startTime).Seconds()
	//fmt.Println("msg: ", timeGap)
	//
	//startTime = time.Now()
	//for i:=0; i<testTimes; i++ {
	//	ownMarshalTest(tx)
	//}
	//timeGap = time.Since(startTime).Seconds()
	//fmt.Println("own: ", timeGap)

	msgpTest(tx)
	ownMarshalTest(tx)

}

func msgpTest(tx Tx) {
	//txMsgpData, _ := tx.MarshalMsg(nil)
	//
	//newTx := &Tx{}
	//newTx.UnmarshalMsg(txMsgpData)

	txMsgpData, err := tx.MarshalMsg(nil)
	if err != nil {
		fmt.Println("marshal error: ", err)
		return
	}
	fmt.Println("msg size: ", len(txMsgpData))
	fmt.Println("msg byte: ", fmt.Sprintf("%x", txMsgpData))

	newTx := &Tx{}
	if _, err := newTx.UnmarshalMsg(txMsgpData); err != nil {
		fmt.Println("unmarshal error: ", err)
		return
	}
	fmt.Println("msg: ", newTx.Dump())
}

func ownMarshalTest(tx Tx) {
	//txBytes := marshalTx(tx)
	//unmarshalTx(txBytes)

	txBytes := marshalTx(tx)
	fmt.Println("own size: ", len(txBytes))
	fmt.Println("own byte: ", fmt.Sprintf("%x", txBytes))

	newTx := unmarshalTx(txBytes)
	fmt.Println("own: ", newTx.Dump())
}

func marshalTx(tx Tx) []byte {
	// init bytes and append 2 bytes for int16 total size
	b := make([]byte, 2)

	// marshal tx base
	b = marshalTxBase(tx.GetBase(), b)

	// marshal tx

	// from - [20]byte
	paramSize := int16(len(tx.From.Bytes))
	sizeBytes := make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, tx.From.Bytes[:]...)

	// to - [20]byte
	paramSize = int16(len(tx.To.Bytes))
	sizeBytes = make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, tx.To.Bytes[:]...)

	// value - big int
	paramSize = int16(len(tx.Value.GetBytes()))
	sizeBytes = make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, tx.Value.GetBytes()...)

	// data - []byte
	paramSize = int16(len(tx.Data))
	sizeBytes = make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, tx.Data...)

	totalSize := int16(len(b))
	common2.SetInt16(b, 0, totalSize)

	return b
}

func unmarshalTx(b []byte) *Tx {
	tx := &Tx{}

	totalSize := common2.GetInt16(b, 0)
	if totalSize != int16(len(b)) {
		panic("total size not correct")
	}
	b = b[2:]

	// tx base
	base, b := unmarshalTxBase(b)
	tx.TxBase = *base

	// from - [20]byte
	paramSize := common2.GetInt16(b, 0)
	b = b[2:]
	from := common.BytesToAddress(b[:paramSize])
	tx.From = &from
	b = b[paramSize:]

	// to - [20]byte
	paramSize = common2.GetInt16(b, 0)
	b = b[2:]
	tx.To = common.BytesToAddress(b[:paramSize])
	b = b[paramSize:]

	// value - big int
	paramSize = common2.GetInt16(b, 0)
	b = b[2:]
	tx.Value = math.NewBigIntFromBigInt(big.NewInt(0).SetBytes(b[:paramSize]))
	b = b[paramSize:]

	// data - []byte
	paramSize = common2.GetInt16(b, 0)
	b = b[2:]
	tx.Data = b[:paramSize]
	b = b[paramSize:]

	return tx
}

func marshalTxBase(base *TxBase, b []byte) []byte {
	/**
	txBase := TxBase{
		Type:         TxBaseTypeNormal,
		Hash:         hash,
		ParentsHash:  common.Hashes{parentHash},
		AccountNonce: 1,
		Height:       1,
		PublicKey:    pubKey,
		Signature:    sig,
		MineNonce:    0,
		Weight:       0,
		inValid:      false,
		Version:      0,
		verified:     0,
	}
	*/

	// type - int8
	b = append(b, byte(base.Type))

	// hash - [32]byte
	paramSize := int16(len(base.Hash.Bytes))
	sizeBytes := make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)

	b = append(b, sizeBytes...)
	b = append(b, base.Hash.Bytes[:]...)

	// parentsHash - list
	listSize := int8(len(base.ParentsHash))
	b = append(b, byte(listSize))
	for _, parentsHash := range base.ParentsHash {
		paramSize := int16(len(parentsHash.Bytes))
		sizeBytes := make([]byte, 2)
		common2.SetInt16(sizeBytes, 0, paramSize)

		b = append(b, sizeBytes...)
		b = append(b, parentsHash.Bytes[:]...)
	}

	// accountNonce - uint64
	sizeBytes = make([]byte, 8)
	common2.SetUint64(sizeBytes, 0, base.AccountNonce)
	b = append(b, sizeBytes...)

	// height - uint64
	sizeBytes = make([]byte, 8)
	common2.SetUint64(sizeBytes, 0, base.Height)
	b = append(b, sizeBytes...)

	// publicKey - PublicKey ( []byte )
	paramSize = int16(len(base.PublicKey))
	sizeBytes = make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, base.PublicKey...)

	// signature - Bytes ( []byte )
	paramSize = int16(len(base.Signature))
	sizeBytes = make([]byte, 2)
	common2.SetInt16(sizeBytes, 0, paramSize)
	b = append(b, sizeBytes...)
	b = append(b, base.Signature...)

	// mined nonce - uint64
	sizeBytes = make([]byte, 8)
	common2.SetUint64(sizeBytes, 0, base.MineNonce)
	b = append(b, sizeBytes...)

	// weight - uint64
	sizeBytes = make([]byte, 8)
	common2.SetUint64(sizeBytes, 0, base.Weight)
	b = append(b, sizeBytes...)

	// version - byte
	b = append(b, base.Version)

	return b
}

func unmarshalTxBase(b []byte) (*TxBase, []byte) {
	/**
	txBase := TxBase{
		Type:         TxBaseTypeNormal,
		Hash:         hash,
		ParentsHash:  common.Hashes{parentHash},
		AccountNonce: 1,
		Height:       1,
		PublicKey:    pubKey,
		Signature:    sig,
		MineNonce:    0,
		Weight:       0,
		inValid:      false,
		Version:      0,
		verified:     0,
	}
	*/

	base := &TxBase{}

	// type - int8
	base.Type = TxBaseType(b[0])
	b = b[1:]

	// hash - [32]byte
	paramSize := common2.GetInt16(b, 0)
	b = b[2:]
	base.Hash = common.BytesToHash(b[:paramSize])
	b = b[paramSize:]

	// parents hash - list of bytes
	listLen := int8(b[0])
	b = b[1:]
	base.ParentsHash = make([]common.Hash, 0)
	for i := int8(0); i < listLen; i++ {
		paramSize := common2.GetInt16(b, 0)
		b = b[2:]
		pHash := common.BytesToHash(b[:paramSize])
		b = b[paramSize:]

		base.ParentsHash = append(base.ParentsHash, pHash)
	}

	// account nonce - uint64
	base.AccountNonce = common2.GetUint64(b, 0)
	b = b[8:]

	// height - uint64
	base.Height = common2.GetUint64(b, 0)
	b = b[8:]

	// public key - []byte
	paramSize = common2.GetInt16(b, 0)
	b = b[2:]
	base.PublicKey = b[:paramSize]
	b = b[paramSize:]

	// signature - []byte
	paramSize = common2.GetInt16(b, 0)
	b = b[2:]
	base.Signature = b[:paramSize]
	b = b[paramSize:]

	// mine nonce - uint64
	base.MineNonce = common2.GetUint64(b, 0)
	b = b[8:]

	// weight - uint64
	base.MineNonce = common2.GetUint64(b, 0)
	b = b[8:]

	// version - byte
	base.Version = b[0]
	b = b[1:]

	return base, b
}
