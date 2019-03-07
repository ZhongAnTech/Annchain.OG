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
package og

import (
	"fmt"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"sync"
	"math/rand"
)

type TipGenerator interface {
	GetRandomTips(n int) (v []types.Txi)
}

type FIFOTipGenerator struct {
	maxCacheSize int
	upstream     TipGenerator
	fifoRing     []types.Txi
	fifoRingPos  int
	fifoRingFull bool
	mu           sync.Mutex
}

func NewFIFOTIpGenerator(upstream TipGenerator, maxCacheSize int) *FIFOTipGenerator {
	return &FIFOTipGenerator{
		upstream:     upstream,
		maxCacheSize: maxCacheSize,
		fifoRing:     make([]types.Txi, maxCacheSize),
	}
}

func (f *FIFOTipGenerator) GetRandomTips(n int) (v []types.Txi) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upstreamTips := f.upstream.GetRandomTips(n)
	// update fifoRing
	for _, upstreamTip := range upstreamTips {
		duplicated := false
		for i := 0; i < f.maxCacheSize; i ++ {
			if f.fifoRing[i] != nil && f.fifoRing[i].GetTxHash() == upstreamTip.GetTxHash() {
				// duplicate, ignore directly
				duplicated = true
				break
			}
		}
		if !duplicated {
			// advance ring and put it in the fifo ring
			f.fifoRing[f.fifoRingPos] = upstreamTip
			f.fifoRingPos ++
			// round
			if f.fifoRingPos == f.maxCacheSize {
				f.fifoRingPos = 0
				f.fifoRingFull = true
			}
		}
	}
	// randomly pick n from fifo cache
	randIndices := make(map[int]bool)
	ringSize := f.fifoRingPos
	if f.fifoRingFull {
		ringSize = f.maxCacheSize
	}
	pickSize := n
	if !f.fifoRingFull && f.fifoRingPos < n {
		pickSize = f.fifoRingPos
	}

	for len(randIndices) != pickSize {
		randIndices[rand.Intn(ringSize)] = true
	}

	// dump those txs
	for k := range randIndices {
		v = append(v, f.fifoRing[k])
	}
	return

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
	GraphVerifier      Verifier     // To verify the graph structure
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

func (m *TxCreator) NewTxWithSeal(from types.Address, to types.Address, value *math.BigInt, data []byte,
	nonce uint64, pubkey crypto.PublicKey, sig crypto.Signature) (tx types.Txi, err error) {
	tx = &types.Tx{
		From: from,
		// TODO
		// should consider the case that to is nil. (contract creation)
		To:    to,
		Value: value,
		Data:  data,
		TxBase: types.TxBase{
			AccountNonce: nonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	tx.GetBase().Signature = sig.Bytes
	tx.GetBase().PublicKey = pubkey.Bytes

	if ok := m.SealTx(tx); !ok {
		logrus.Warn("failed to seal tx")
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")

	return tx, nil
}

func (m *TxCreator) NewSignedTx(from types.Address, to types.Address, value *math.BigInt, accountNonce uint64,
	privateKey crypto.PrivateKey) types.Txi {
	if privateKey.Type != m.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.NewUnsignedTx(from, to, value, accountNonce)
	// do sign work
	signature := m.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = m.Signer.PubKey(privateKey).Bytes
	return tx
}

func (m *TxCreator) NewUnsignedSequencer(issuer types.Address, Height uint64, accountNonce uint64) types.Txi {
	tx := types.Sequencer{
		Issuer: issuer,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
			Height:       Height,
		},
	}
	return &tx
}

func (m *TxCreator) NewSignedSequencer(issuer types.Address, height uint64, accountNonce uint64, privateKey crypto.PrivateKey) types.Txi {
	if privateKey.Type != m.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.NewUnsignedSequencer(issuer, height, accountNonce)
	// do sign work
	signature := m.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = m.Signer.PubKey(privateKey).Bytes
	return tx
}

// validateGraphStructure validates if parents are not conflicted, not double spending or other misbehaviors
func (m *TxCreator) validateGraphStructure(parents []types.Txi) (ok bool) {
	ok = true
	for _, parent := range parents {
		ok = ok && m.GraphVerifier.Verify(parent)
		if !ok {
			return
		}
	}
	return
}

func (m *TxCreator) tryConnect(tx types.Txi, parents []types.Txi) (txRet types.Txi, ok bool) {
	parentHashes := make([]types.Hash, len(parents))
	for i, parent := range parents {
		parentHashes[i] = parent.GetTxHash()
	}

	//calculate weight
	tx.GetBase().Weight = tx.CalculateWeight(parents)

	tx.GetBase().ParentsHash = parentHashes
	// verify if the hash of the structure meet the standard.
	hash := tx.CalcTxHash()
	if hash.Cmp(m.MaxTxHash) < 0 {
		tx.GetBase().Hash = hash
		logrus.WithField("hash", hash).WithField("parent", tx.Parents()).Trace("new tx connected")
		// yes
		txRet = tx
		//ok = m.validateGraphStructure(parents)
		ok = m.GraphVerifier.Verify(tx)
		if !ok {
			logrus.Debug("NOT OK")
		}
		logrus.WithFields(logrus.Fields{
			"tx": tx,
			"ok": ok,
		}).Trace("validate graph structure for tx being connected")
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
		mineCount++
		go m.Miner.StartMine(tx, m.MaxMinedHash, minedNonce+1, respChan)
		select {
		case minedNonce = <-respChan:
			tx.GetBase().MineNonce = minedNonce // Actually, this value is already set during mining.
			//logrus.Debugf("Total time for Mining: %d ns, %d times", time.Since(timeStart).Nanoseconds(), minedNonce)
			// pick up parents.
			for i := 0; i < m.MaxConnectingTries; i++ {
				connectionTries++
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
		"elapsedns":  time.Since(timeStart).Nanoseconds(),
		"re-mine":    mineCount,
		"nonce":      minedNonce,
		"re-connect": connectionTries,
	}).Debugf("total time for mining")
	return true
}
