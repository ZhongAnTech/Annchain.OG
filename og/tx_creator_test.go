package og

import (
	"testing"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/og/miner"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
	"time"
	"github.com/annchain/OG/common/math"
)



func Init() *TxCreator {
	txc := TxCreator{
		Signer:             &crypto.SignerEd25519{},
		TipGenerator:       &dummyTxPoolRandomTx{},
		Miner:              &miner.PoWMiner{},
		MaxConnectingTries: 100,
		MaxTxHash:          types.HexToHash("0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaxMinedHash:       types.HexToHash("0x00000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}
	return &txc
}

func TestTxCreator(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	txc := Init()
	tx := txc.TipGenerator.GetRandomTips(1)[0].(*types.Tx)
	_, priv, err := txc.Signer.RandomKeyPair()
	assert.NoError(t, err)
	time1 := time.Now()
	txSigned := txc.NewSignedTx(tx.From, tx.To, tx.Value, tx.AccountNonce, priv)
	logrus.Infof("total time for Signing: %d ns", time.Since(time1).Nanoseconds())
	ok := txc.SealTx(txSigned)
	logrus.Infof("result: %t %v", ok, txSigned)
}

func TestSequencerCreator(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	txc := Init()
	_, priv, err := txc.Signer.RandomKeyPair()
	assert.NoError(t, err)
	time1 := time.Now()

	// for copy
	randomSeq := types.RandomSequencer()

	txSigned := txc.NewSignedSequencer(randomSeq.Id, randomSeq.ContractHashOrder, randomSeq.AccountNonce, priv)
	logrus.Infof("total time for Signing: %d ns", time.Since(time1).Nanoseconds())
	ok := txc.SealTx(txSigned)
	logrus.Infof("result: %t %v", ok, txSigned)
}

func sampleTxi(selfHash string, parentsHash []string, baseType types.TxBaseType) types.Txi {

	tx := &types.Tx{TxBase: types.TxBase{
		ParentsHash: []types.Hash{},
		Type:        types.TxBaseTypeNormal,
		Hash:        types.HexToHash(selfHash),
	},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, types.HexToHash(h))
	}
	return tx
}


func TestBuildDag(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	pool := &DummyTxPoolMiniTx{}
	pool.Init()
	txc := TxCreator{
		Signer:             &crypto.SignerEd25519{},
		TipGenerator:       pool,
		Miner:              &miner.PoWMiner{},
		MaxConnectingTries: 10,
		MaxTxHash:          types.HexToHash("0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaxMinedHash:       types.HexToHash("0x000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}

	_, privateKey, _ := txc.Signer.RandomKeyPair()

	txs := []types.Txi{
		txc.NewSignedSequencer(0, []types.Hash{}, 0, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x01"), types.HexToAddress("0x02"), math.NewBigInt(10), 0, privateKey),
		txc.NewSignedSequencer(1, []types.Hash{}, 1, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x02"), types.HexToAddress("0x03"), math.NewBigInt(9), 0, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x03"), types.HexToAddress("0x04"), math.NewBigInt(8), 0, privateKey),
		txc.NewSignedSequencer(2, []types.Hash{}, 2, privateKey),
	}

	txs[0].GetBase().Hash = txs[0].CalcTxHash()
	pool.Add(txs[0])
	for i := 1 ; i< len(txs) ; i++{
		if ok := txc.SealTx(txs[i]); ok{
			pool.Add(txs[i])
		}
	}
}
