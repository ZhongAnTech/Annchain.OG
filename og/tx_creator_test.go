package og

import (
	"testing"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/og/miner"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []types.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, types.RandomTx())
	}
	return
}

func Init() *TxCreator {
	txc := TxCreator{
		Signer:           &crypto.SignerEd25519{},
		TipGenerator:     &dummyTxPoolRandomTx{},
		Miner:            &miner.PoWMiner{},
		MaxMinedHash:     types.HexToHash("0x00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaxStructureHash: types.HexToHash("0x00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}
	return &txc
}

func TestTxCreator(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	txc := Init()
	tx := txc.TipGenerator.GetRandomTips(1)[0].(*types.Tx)
	_, priv, err := txc.Signer.RandomKeyPair()
	assert.NoError(t, err)
	txSigned := txc.NewSignedTx(tx.From, tx.To, tx.Value,tx.AccountNonce, priv)
	ok := txc.SealTx(txSigned)
	logrus.Infof("Result: %t %v", ok , txSigned)
	// TODO: Fix 4 parents
}
