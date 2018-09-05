package og

import (
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/common/crypto"
	"time"
	"github.com/annchain/OG/og/miner"
	"github.com/sirupsen/logrus"
)

type TipGenerator interface {
	GetRandomTips(n int) (v []types.Txi)
}

// TxCreator creates tx and do the signing and mining
type TxCreator struct {
	Signer           crypto.Signer
	Miner            miner.Miner
	TipGenerator     TipGenerator // usually tx_pool
	MaxStructureHash types.Hash   // The difficultiy of strctureHash
	MaxMinedHash     types.Hash   // The difficultiy of minedHash
}

func (m *TxCreator) NewUnsignedTx(from types.Address, to types.Address, value *math.BigInt, accountNonce uint64) types.Txi {
	tx := types.Tx{
		Value: value,
		To:    to,
		From:  from,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	return &tx
}

func (m *TxCreator) NewSignedTx(from types.Address, to types.Address, value *math.BigInt, accountNonce uint64,
	privateKey crypto.PrivateKey) types.Txi {
	tx := m.NewUnsignedTx(from, to, value, accountNonce)
	// do sign work
	signature := m.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = m.Signer.PubKey(privateKey).Bytes
	return tx
}

func (m *TxCreator) tryConnect(tx types.Txi, parents []types.Txi) (txRet types.Txi, ok bool) {
	parentHashes := make([]types.Hash, len(parents))
	for _, parent := range parents {
		parentHashes = append(parentHashes, parent.MinedHash())
	}

	tx.GetBase().ParentsHash = parentHashes
	// verify if the hash of the structure meet the standard.
	if tx.StructureHash().Cmp(m.MaxStructureHash) < 0 {
		logrus.Debugf("Connected %s %s", tx.StructureHash().Hex(), m.MaxStructureHash.Hex())
		logrus.Debugf("Parents: %s %s", tx.GetBase().ParentsHash[0].Hex(),tx.GetBase().ParentsHash[1].Hex())
		// yes
		txRet = tx
		return txRet, true
	} else {
		logrus.Debugf("Failed to connected %s %s", tx.StructureHash().Hex(), m.MaxStructureHash.Hex())
		return nil, false
	}
}

// SealTx constantly queries pool to get 2 txs that could leads to a proper hash.
func (m *TxCreator) SealTx(tx types.Txi) (ok bool) {
	for {
		// TODO: What if there is not enough tips in the pool? (Deadlock waiting)
		txs := m.TipGenerator.GetRandomTips(2)
		logrus.Debugf("Got 2 tips: %s %s", txs[0].MinedHash().Hex(), txs[1].MinedHash().Hex())
		if len(txs) == 0 {
			// no enough tips, sleep a while
			time.Sleep(time.Second)
			continue
		}
		if _, ok := m.tryConnect(tx, txs); ok {
			break
		}
	}
	// connected, calc nonce
	respChan := make(chan uint64)
	go m.Miner.StartMine(tx, m.MaxMinedHash, respChan)
	select {
	case nonce := <-respChan:
		tx.SetMineNonce(nonce) // Actually, this value is already set during mining.
		return true
	case <-time.NewTimer(time.Minute * 5).C:
		return false
	}

}
