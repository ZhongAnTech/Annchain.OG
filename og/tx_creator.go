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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/types/tx_types"
	"sync/atomic"
	"time"

	"math/rand"
	"sync"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type TipGenerator interface {
	GetRandomTips(n int) (v []types.Txi)
}

type GetStateRoot interface {
	PreConfirm(seq *tx_types.Sequencer) (hash common.Hash, err error)
}

type FIFOTipGenerator struct {
	maxCacheSize int
	upstream     TipGenerator
	fifoRing     []types.Txi
	fifoRingPos  int
	fifoRingFull bool
	mu           sync.RWMutex
}

func NewFIFOTIpGenerator(upstream TipGenerator, maxCacheSize int) *FIFOTipGenerator {
	return &FIFOTipGenerator{
		upstream:     upstream,
		maxCacheSize: maxCacheSize,
		fifoRing:     make([]types.Txi, maxCacheSize),
	}
}

func (f *FIFOTipGenerator) validation() {
	var k int
	for i := 0; i < f.maxCacheSize; i++ {
		if f.fifoRing[i] == nil {
			if f.fifoRingPos > i {
				f.fifoRingPos = i
			}
			break
		}
		//if the tx is became fatal tx ,remove it from tips
		if f.fifoRing[i].InValid() {
			logrus.WithField("tx ", f.fifoRing[i]).Debug("found invalid tx")
			if !f.fifoRingFull {
				if f.fifoRingPos > 0 {
					if f.fifoRingPos-1 == i {
						f.fifoRing[i] = nil
					} else if f.fifoRingPos-1 > i {
						f.fifoRing[i] = f.fifoRing[f.fifoRingPos-1]
					} else {
						f.fifoRing[i] = nil
						break
					}
					f.fifoRingPos--
				}
			} else {
				f.fifoRing[i] = f.fifoRing[f.maxCacheSize-k-1]
				f.fifoRing[f.maxCacheSize-k-1] = nil
				i--
				k++
				f.fifoRingFull = false
			}
		}
	}
}

func (f *FIFOTipGenerator) GetRandomTips(n int) (v []types.Txi) {
	f.mu.Lock()
	defer f.mu.Unlock()
	upstreamTips := f.upstream.GetRandomTips(n)
	//checkValidation
	f.validation()
	// update fifoRing
	for _, upstreamTip := range upstreamTips {
		duplicated := false
		for i := 0; i < f.maxCacheSize; i++ {
			if f.fifoRing[i] != nil && f.fifoRing[i].GetTxHash() == upstreamTip.GetTxHash() {
				// duplicate, ignore directly
				duplicated = true
				break
			}
		}
		if !duplicated {
			// advance ring and put it in the fifo ring
			f.fifoRing[f.fifoRingPos] = upstreamTip
			f.fifoRingPos++
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
	Miner              miner.Miner
	TipGenerator       TipGenerator // usually tx_pool
	MaxTxHash          common.Hash  // The difficultiy of TxHash
	MaxMinedHash       common.Hash  // The difficultiy of MinedHash
	MaxConnectingTries int          // Max number of times to find a pair of parents. If exceeded, try another nonce.
	DebugNodeId        int          // Only for debug. This value indicates tx sender and is temporarily saved to tx.height
	GraphVerifier      Verifier     // To verify the graph structure
	quit               bool
	archiveNonce       uint64
	NoVerifyMindHash   bool
	NoVerifyMaxTxHash  bool
	GetStateRoot       GetStateRoot
	TxFormatVerifier    TxFormatVerifier
}

func (t *TxCreator) GetArchiveNonce() uint64 {
	return atomic.AddUint64(&t.archiveNonce, 1)
}

func (t *TxCreator) Stop() {
	t.quit = true
}

func (m *TxCreator) NewUnsignedTx(from common.Address, to common.Address, value *math.BigInt, accountNonce uint64, tokenId int32) types.Txi {
	tx := tx_types.Tx{
		Value:   value,
		To:      to,
		From:    &from,
		TokenId: tokenId,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	return &tx
}

func (m *TxCreator) NewArchiveWithSeal(data []byte) (tx types.Txi, err error) {
	tx = &tx_types.Archive{
		TxBase: types.TxBase{
			AccountNonce: m.GetArchiveNonce(),
			Type:         types.TxBaseTypeArchive,
		},
		Data: data,
	}

	if ok := m.SealTx(tx, nil); !ok {
		logrus.Warn("failed to seal tx")
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")

	return tx, nil
}

func (m *TxCreator) NewTxWithSeal(from common.Address, to common.Address, value *math.BigInt, data []byte,
	nonce uint64, pubkey crypto.PublicKey, sig crypto.Signature, tokenId int32) (tx types.Txi, err error) {
	tx = &tx_types.Tx{
		From: &from,
		// TODO
		// should consider the case that to is nil. (contract creation)
		To:      to,
		Value:   value,
		Data:    data,
		TokenId: tokenId,
		TxBase: types.TxBase{
			AccountNonce: nonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	tx.GetBase().Signature = sig.Bytes
	tx.GetBase().PublicKey = pubkey.Bytes

	if ok := m.SealTx(tx, nil); !ok {
		logrus.Warn("failed to seal tx")
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")

	return tx, nil
}

func (m *TxCreator) NewActionTxWithSeal(from common.Address, to common.Address, value *math.BigInt, action byte,
	nonce uint64, enableSpo bool, TokenId int32, tokenName string, pubkey crypto.PublicKey, sig crypto.Signature) (tx types.Txi, err error) {
	tx = &tx_types.ActionTx{
		From: &from,
		// TODO
		// should consider the case that to is nil. (contract creation)
		TxBase: types.TxBase{
			AccountNonce: nonce,
			Type:         types.TxBaseAction,
		},
		Action: action,
		ActionData: &tx_types.PublicOffering{
			Value:     value,
			EnableSPO: enableSpo,
			TokenId:   TokenId,
			TokenName: tokenName,
		},
	}
	tx.GetBase().Signature = sig.Bytes
	tx.GetBase().PublicKey = pubkey.Bytes

	if ok := m.SealTx(tx, nil); !ok {
		logrus.Warn("failed to seal tx")
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	tx.SetVerified(types.VerifiedFormat)
	return tx, nil
}

func (m *TxCreator) NewSignedTx(from common.Address, to common.Address, value *math.BigInt, accountNonce uint64,
	privateKey crypto.PrivateKey, tokenId int32) types.Txi {
	if privateKey.Type != crypto.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.NewUnsignedTx(from, to, value, accountNonce, tokenId)
	// do sign work
	signature := crypto.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = crypto.Signer.PubKey(privateKey).Bytes
	return tx
}

func (m *TxCreator) NewUnsignedSequencer(issuer common.Address, Height uint64, accountNonce uint64) *tx_types.Sequencer {
	tx := tx_types.Sequencer{
		Issuer: &issuer,
		TxBase: types.TxBase{
			AccountNonce: accountNonce,
			Type:         types.TxBaseTypeSequencer,
			Height:       Height,
		},
	}
	return &tx
}

func (m *TxCreator) NewSignedSequencer(issuer common.Address, height uint64, accountNonce uint64, privateKey crypto.PrivateKey) types.Txi {
	if privateKey.Type != crypto.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.NewUnsignedSequencer(issuer, height, accountNonce)
	// do sign work
	logrus.Tracef("seq before sign, the sign type is: %s", crypto.Signer.GetCryptoType().String())
	signature := crypto.Signer.Sign(privateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = crypto.Signer.PubKey(privateKey).Bytes
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

func (m *TxCreator) tryConnect(tx types.Txi, parents []types.Txi, privateKey *crypto.PrivateKey) (txRet types.Txi, ok bool) {
	parentHashes := make(common.Hashes, len(parents))
	for i, parent := range parents {
		parentHashes[i] = parent.GetTxHash()
	}
	//calculate weight
	tx.GetBase().Weight = tx.CalculateWeight(parents)

	tx.GetBase().ParentsHash = parentHashes
	// verify if the hash of the structure meet the standard.
	hash := tx.CalcTxHash()
	if m.NoVerifyMaxTxHash || hash.Cmp(m.MaxTxHash) < 0 {
		tx.GetBase().Hash = hash
		logrus.WithField("hash", hash).WithField("parent", tx.Parents()).Trace("new tx connected")
		// yes
		txRet = tx
		//ok = m.validateGraphStructure(parents)
		//todo why verify here duplicated verification
		ok = m.GraphVerifier.Verify(tx)
		if !ok {
			logrus.WithField("tx ",tx).Debug("NOT OK")
			return txRet, ok
		}
		tx.SetVerified(types.VerifiedGraph)
		//ok = true
		logrus.WithFields(logrus.Fields{
			"tx": tx,
			"ok": ok,
		}).Trace("validate graph structure for tx being connected")

		if tx.GetType() == types.TxBaseTypeSequencer {
			tx.GetBase().Signature = crypto.Signer.Sign(*privateKey, tx.SignatureTargets()).Bytes
			tx.GetBase().Hash = tx.CalcTxHash()
		}

		return txRet, ok
	} else {
		//logrus.Debugf("Failed to connected %s %s", hash.Hex(), m.MaxTxHash.Hex())
		return nil, false
	}
}

// SealTx do mining first, then pick up parents from tx pool which could leads to a proper hash.
// If there is no proper parents, Mine again.
func (m *TxCreator) SealTx(tx types.Txi, priveKey *crypto.PrivateKey) (ok bool) {
	// record the mining times.
	mineCount := 0
	connectionTries := 0
	minedNonce := uint64(0)

	timeStart := time.Now()
	respChan := make(chan uint64)
	defer close(respChan)
	done := false
	for !done {
		if m.quit {
			logrus.Info("got quit signal")
			return false
		}
		mineCount++
		if !m.NoVerifyMindHash {
			goroutine.New(func() {
				m.Miner.StartMine(tx, m.MaxMinedHash, minedNonce+1, respChan)
			})
		} else {
			goroutine.New(func() {
				respChan <- 1
			})
		}
		select {
		case minedNonce = <-respChan:
			tx.GetBase().MineNonce = minedNonce // Actually, this value is already set during mining.
			//logrus.Debugf("Total time for Mining: %d ns, %d times", time.Since(timeStart).Nanoseconds(), minedNonce)
			// pick up parents.
			for i := 0; i < m.MaxConnectingTries; i++ {
				if m.quit {
					logrus.Info("got quit signal")
					return false
				}
				connectionTries++
				txs := m.TipGenerator.GetRandomTips(2)

				//logrus.Debugf("Got %d Tips: %s", len(txs), common.HashesToString(tx.Parents()))
				if len(txs) == 0 {
					// Impossible. At least genesis is there
					logrus.Warn("at least genesis is there. Wait for loading")
					time.Sleep(time.Second * 2)
					continue
				}

				if _, ok := m.tryConnect(tx, txs, priveKey); ok {
					done = true
					break
				}else {
					logrus.WithField("parents ",txs).WithField("connection tries ", connectionTries).WithField("tx ",tx).Debug("NOT OK")
				}
			}
			if mineCount > 1 {
				return false
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

func (m *TxCreator) GenerateSequencer(issuer common.Address, Height uint64, accountNonce uint64, privateKey *crypto.PrivateKey, blsPubKey []byte) (seq *tx_types.Sequencer) {
	tx := m.NewUnsignedSequencer(issuer, Height, accountNonce)
	//for sequencer no mined nonce
	// record the mining times.
	pubkey := crypto.Signer.PubKey(*privateKey)
	tx.GetBase().PublicKey = pubkey.Bytes
	tx.SetSender(pubkey.Address())
	if blsPubKey != nil {
		// proposed by bft
		tx.BlsJointPubKey = blsPubKey
		tx.Proposing = true
	}
	// else it is proposed by delegate for solo

	connectionTries := 0
	timeStart := time.Now()
	//logrus.Debugf("Total time for Mining: %d ns, %d times", time.Since(timeStart).Nanoseconds(), minedNonce)
	// pick up parents.
	var ok bool
	for connectionTries = 0; connectionTries < m.MaxConnectingTries; connectionTries++ {
		if m.quit {
			logrus.Info("got quit signal")
			return tx
		}
		parents := m.TipGenerator.GetRandomTips(2)

		//logrus.Debugf("Got %d Tips: %s", len(txs), common.HashesToString(tx.Parents()))
		if len(parents) == 0 {
			// Impossible. At least genesis is there
			logrus.Warn("at least genesis is there. Wait for loading")
			time.Sleep(time.Second * 1)
			continue
		}
		parentHashes := make(common.Hashes, len(parents))
		for i, parent := range parents {
			parentHashes[i] = parent.GetTxHash()
		}

		//calculate weight
		tx.GetBase().Weight = tx.CalculateWeight(parents)
		tx.GetBase().ParentsHash = parentHashes
		// verify if the hash of the structure meet the standard.
		logrus.WithField("id ", tx.GetHeight()).WithField("parent", tx.Parents()).Trace("new tx connected")
		//ok = m.validateGraphStructure(parents)
		ok = m.GraphVerifier.Verify(tx)
		if !ok {
			logrus.Debug("NOT OK")
			logrus.WithFields(logrus.Fields{
				"tx": tx,
				"ok": ok,
			}).Trace("validate graph structure for tx being connected")
			continue
		} else {
			//calculate root
			//calculate signatrue
			root, err := m.GetStateRoot.PreConfirm(tx)
			if err != nil {
				logrus.WithField("seq ", tx).Errorf("CalculateStateRoot err  %v", err)
				return nil
			}
			tx.StateRoot = root
			tx.GetBase().Signature = crypto.Signer.Sign(*privateKey, tx.SignatureTargets()).Bytes
			tx.GetBase().Hash = tx.CalcTxHash()
			tx.SetVerified(types.VerifiedGraph)
			tx.SetVerified(types.VerifiedFormat)
			break
		}
	}
	if ok {
		logrus.WithFields(logrus.Fields{
			"elapsedns":  time.Since(timeStart).Nanoseconds(),
			"re-connect": connectionTries,
		}).Tracef("total time for mining")
		return tx
	}
	logrus.WithFields(logrus.Fields{
		"elapsedns":  time.Since(timeStart).Nanoseconds(),
		"re-connect": connectionTries,
	}).Warnf("generate sequencer failed")
	return nil
}
