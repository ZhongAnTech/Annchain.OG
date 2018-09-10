package core_test

// TODO

import (
	"testing"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/common/crypto"
)

func generateTxPool() (*core.TxPool, *crypto.PrivateKey) {
	txpoolconfig := core.TxPoolConfig{
		QueueSize: 100,
		TipsSize: 100,
		ResetDuration: 5,
		TxVerifyTime: 2,
		TxValidTime: 7,
	}

	core.NewTxPool(txpoolconfig)

	return nil, nil
}

func TestBadTransaction(t *testing.T) {

}



