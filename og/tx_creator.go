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
	Signer             crypto.Signer
	Miner              miner.Miner
	TipGenerator       TipGenerator // usually tx_pool
	MaxTxHash          types.Hash   // The difficultiy of TxHash
	MaxMinedHash       types.Hash   // The difficultiy of MinedHash
	MaxConnectingTries int          // Max number of times to find a pair of parents. If exceeded, try another nonce.
	DebugNodeId        int          // Only for debug. This value indicates tx sender and is temporarily saved to tx.height
}

func (m *TxCreator) NewUnsignedTx(from types.Address, to types.Address, value *math.BigInt, accountNonce uint64) types.Txi {
	tx := types.Tx{
		Value: value,
		To:    to,
		From:  from,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeNormal,
			Height:       uint64(m.DebugNodeId),
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

func (m *TxCreator) NewUnsignedSequencer(id uint64, contractHashOrder []types.Hash, accountNonce uint64) types.Txi {
	tx := types.Sequencer{
		Id:                id,
		ContractHashOrder: contractHashOrder,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
			Height:       uint64(m.DebugNodeId),
		},
	}
	return &tx
}

func (m *TxCreator) NewSignedSequencer(id uint64, contractHashOrder []types.Hash, accountNonce uint64, privateKey crypto.PrivateKey) types.Txi {
	tx := m.NewUnsignedSequencer(id, contractHashOrder, accountNonce)
	// do sign work
	signature := m.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = m.Signer.PubKey(privateKey).Bytes
	return tx
}

// validateGraphStructure validates if parents are not conflicted, not double spending or other misbehaviors
// TODO: fill this.
func (m *TxCreator) validateGraphStructure(parents []types.Txi) (ok bool) {
	return true
}

func (m *TxCreator) tryConnect(tx types.Txi, parents []types.Txi) (txRet types.Txi, ok bool) {
	parentHashes := make([]types.Hash, len(parents))
	for i, parent := range parents {
		parentHashes[i] = parent.GetTxHash()
	}

	tx.GetBase().ParentsHash = parentHashes
	// verify if the hash of the structure meet the standard.
	hash := tx.CalcTxHash()
	if hash.Cmp(m.MaxTxHash) < 0 {
		tx.GetBase().Hash = hash
		logrus.WithField("hash", hash).WithField("parent", types.HashesToString(tx.Parents())).Debug("new tx connected")
		// yes
		txRet = tx
		ok = m.validateGraphStructure(parents)
		logrus.WithFields(logrus.Fields{
			"tx": tx,
			"ok": ok,
		}).Debugf("validate graph structure")
		return txRet, ok
	} else {
		//logrus.Debugf("Failed to connected %s %s", hash.Hex(), m.MaxTxHash.Hex())
		return nil, false
	}
}

// SealTx do mining first, then pick up parents from tx pool which could leads to a proper hash.
// If there is no proper parents, Mine again.
func (m *TxCreator) SealTx(tx types.Txi) (ok bool) {
	// record the mining times.
	mineCount := 0
	connectionTries := 0
	minedNonce := uint64(0)

	timeStart := time.Now()
	respChan := make(chan uint64)
	defer close(respChan)
	done := false
	for !done {
		mineCount ++
		go m.Miner.StartMine(tx, m.MaxMinedHash, minedNonce+1, respChan)
		select {
		case minedNonce = <-respChan:
			tx.GetBase().MineNonce = minedNonce // Actually, this value is already set during mining.
			//logrus.Debugf("Total time for Mining: %d ns, %d times", time.Since(timeStart).Nanoseconds(), minedNonce)
			// pick up parents.
			for i := 0; i < m.MaxConnectingTries; i++ {
				connectionTries ++
				txs := m.TipGenerator.GetRandomTips(2)

				//logrus.Debugf("Got %d Tips: %s", len(txs), types.HashesToString(tx.Parents()))
				if len(txs) == 0 {
					// Impossible. At least genesis is there
					logrus.Warn("at least genesis is there. Wait for loading")
					time.Sleep(time.Second * 2)
					continue
				}

				if _, ok := m.tryConnect(tx, txs); ok {
					done = true
					break
				}
			}
		case <-time.NewTimer(time.Minute * 5).C:
			return false
		}
	}
	logrus.WithFields(logrus.Fields{
		"elapsedns": time.Since(timeStart).Nanoseconds(),
		"re-mine": mineCount,
		"nonce": minedNonce,
		"re-connect": connectionTries,
	}).Debugf("total time for mining")
	return true
}
