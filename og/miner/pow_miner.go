// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package miner

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/byteutil"
	"github.com/annchain/OG/og/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"math"
)

type PoWMiner struct {
}

func (m *PoWMiner) Mine(tx types.Txi, targetMax common.Hash, start uint64, responseChan chan uint64) bool {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.MineTx(tx.(*types.Tx), targetMax, start, responseChan)
	case types.TxBaseTypeSequencer:
		return m.MineSequencer(tx.(*types.Sequencer), targetMax, start, responseChan)
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) MineTx(tx *types.Tx, targetMax common.Hash, start uint64, responseChan chan uint64) bool {
	var i uint64
	for i = start; i <= math.MaxUint64; i++ {
		tx.SetMineNonce(i)
		//logrus.Debugf("%10d %s %s", i, tx.Hash().Hex(), targetMax.Hex())
		if m.CalcMinedHashTx(tx).Cmp(targetMax) < 0 {
			//logrus.Debugf("Hash found: %s with %d", tx.CalcMinedHash().Hex(), i)
			responseChan <- i
			return true
		}
	}
	return false
}

func (m *PoWMiner) MineSequencer(seq *types.Sequencer, targetMax common.Hash, start uint64, responseChan chan uint64) bool {
	var i uint64
	for i = start; i <= math.MaxUint64; i++ {
		seq.SetMineNonce(i)
		//logrus.Debugf("%10d %s %s", i, tx.Hash().Hex(), targetMax.Hex())
		if m.CalcMinedHashSequencer(seq).Cmp(targetMax) < 0 {
			//logrus.Debugf("Hash found: %s with %d", tx.CalcMinedHash().Hex(), i)
			responseChan <- i
			return true
		}
	}
	return false
}

func (m *PoWMiner) CalcHash(tx types.Txi) (hash common.Hash) {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.CalcHashTx(tx.(*types.Tx))
	case types.TxBaseTypeSequencer:
		return m.CalcHashSequencer(tx.(*types.Sequencer))
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) CalcHashTx(tx *types.Tx) (hash common.Hash) {
	w := byteutil.NewBinaryWriter()

	for _, ancestor := range tx.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	//w.Write(m.Weight)
	mineHash := m.CalcMinedHashTx(tx)
	w.Write(mineHash.Bytes)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (m *PoWMiner) CalcHashSequencer(seq *types.Sequencer) (hash common.Hash) {
	// TODO: double check the hash content
	w := byteutil.NewBinaryWriter()

	for _, ancestor := range seq.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	//w.Write(m.Weight)
	w.Write(seq.Signature)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (m *PoWMiner) CalcMinedHash(tx types.Txi) (hash common.Hash) {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.CalcMinedHashTx(tx.(*types.Tx))
	case types.TxBaseTypeSequencer:
		return m.CalcMinedHashSequencer(tx.(*types.Sequencer))
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) CalcMinedHashTx(tx *types.Tx) (hash common.Hash) {
	w := byteutil.NewBinaryWriter()
	//if !CanRecoverPubFromSig {
	w.Write(tx.PublicKey.ToBytes())
	//}
	w.Write(tx.Signature.ToBytes(), tx.MineNonce)
	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (m *PoWMiner) CalcMinedHashSequencer(seq *types.Sequencer) (hash common.Hash) {
	w := byteutil.NewBinaryWriter()

	for _, ancestor := range seq.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	//w.Write(s.Weight)
	w.Write(seq.Signature)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (m *PoWMiner) IsMineHashValid(tx types.Txi, mineHash common.Hash, targetHashMax common.Hash) bool {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.IsMineHashValidForTx(tx.(*types.Tx), mineHash, targetHashMax)
	case types.TxBaseTypeSequencer:
		return m.IsMineHashValidForSequencer(tx.(*types.Sequencer), mineHash, targetHashMax)
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) IsMineHashValidForTx(tx *types.Tx, mineHash common.Hash, targetHashMax common.Hash) bool {
	hash := m.CalcMinedHashTx(tx)
	if hash.Cmp(mineHash) != 0 {
		logrus.WithField("should", hash).WithField("actual", mineHash).Warn("hash is fake")
		return false
	}
	if hash.Cmp(targetHashMax) >= 0 {
		logrus.WithField("should", targetHashMax).WithField("actual", mineHash).Warn("hash is too large")
		return false
	}
	return true
}

func (m *PoWMiner) IsMineHashValidForSequencer(tx *types.Sequencer, mineHash common.Hash, targetHashMax common.Hash) bool {
	hash := m.CalcMinedHashSequencer(tx)
	if hash.Cmp(mineHash) != 0 {
		logrus.WithField("should", hash).WithField("actual", mineHash).Warn("hash is fake")
		return false
	}
	if hash.Cmp(targetHashMax) >= 0 {
		logrus.WithField("should", targetHashMax).WithField("actual", mineHash).Warn("hash is too large")
		return false
	}
	return true
}

func (m *PoWMiner) IsHashValid(tx types.Txi, hash common.Hash, targetHashMax common.Hash) bool {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.IsHashValidForTx(tx.(*types.Tx), hash, targetHashMax)
	case types.TxBaseTypeSequencer:
		return m.IsHashValidForSequencer(tx.(*types.Sequencer), hash, targetHashMax)
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) IsHashValidForTx(tx *types.Tx, hash common.Hash, targetHashMax common.Hash) bool {
	calcHash := m.CalcHashTx(tx)
	if calcHash.Cmp(hash) != 0 {
		logrus.WithField("should", calcHash).WithField("actual", hash).Warn("hash is fake")
		return false
	}
	if calcHash.Cmp(targetHashMax) >= 0 {
		logrus.WithField("should", targetHashMax).WithField("actual", hash).Warn("hash is too large")
		return false
	}
	return true
}

func (m *PoWMiner) IsHashValidForSequencer(tx *types.Sequencer, hash common.Hash, targetHashMax common.Hash) bool {
	calcHash := m.CalcHashSequencer(tx)
	if calcHash.Cmp(hash) != 0 {
		logrus.WithField("should", calcHash).WithField("actual", hash).Warn("calcHash is fake")
		return false
	}
	if calcHash.Cmp(targetHashMax) >= 0 {
		logrus.WithField("should", targetHashMax).WithField("actual", hash).Warn("calcHash is too large")
		return false
	}
	return true
}

func (m *PoWMiner) IsGood(tx types.Txi, targetMineHashMax common.Hash, targetHashMax common.Hash) bool {
	switch tx.GetType() {
	case types.TxBaseTypeTx:
		return m.IsGoodTx(tx.(*types.Tx), targetMineHashMax, targetHashMax)
	case types.TxBaseTypeSequencer:
		return m.IsGoodSequencer(tx.(*types.Sequencer), targetMineHashMax, targetHashMax)
	default:
		panic("tx type not supported")
	}
}

func (m *PoWMiner) IsGoodTx(tx *types.Tx, targetMineHashMax common.Hash, targetHashMax common.Hash) bool {
	return m.IsMineHashValidForTx(tx, tx.Hash, targetMineHashMax) && m.IsHashValidForTx(tx, tx.Hash, targetHashMax)
}

func (m *PoWMiner) IsGoodSequencer(seq *types.Sequencer, targetMineHashMax common.Hash, targetHashMax common.Hash) bool {
	return m.IsMineHashValidForSequencer(seq, seq.Hash, targetMineHashMax) && m.IsHashValidForSequencer(seq, seq.Hash, targetHashMax)
}
