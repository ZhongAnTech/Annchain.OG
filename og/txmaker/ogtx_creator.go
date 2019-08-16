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
package txmaker

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/protocol"
	"github.com/annchain/OG/types/tx_types"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type TipGenerator interface {
	GetRandomTips(n int) (v []types.Txi)
	GetByNonce(addr common.Address, nonce uint64) types.Txi
	IsBadSeq(seq *tx_types.Sequencer) error
}

type StateRootProvider interface {
	PreConfirm(seq *tx_types.Sequencer) (hash common.Hash, err error)
}

// OGTxCreator creates tx and do the signing and mining for OG.
type OGTxCreator struct {
	Miner              miner.Miner
	TipGenerator       TipGenerator      // usually tx_pool
	MaxTxHash          common.Hash       // The difficultiy of TxHash
	MaxMinedHash       common.Hash       // The difficultiy of MinedHash
	MaxConnectingTries int               // Max number of times to find a pair of parents. If exceeded, try another nonce.
	DebugNodeId        int               // Only for debug. This value indicates tx sender and is temporarily saved to tx.height
	GraphVerifier      protocol.Verifier // To verify the graph structure
	quit               bool
	archiveNonce       uint64
	NoVerifyMindHash   bool
	NoVerifyMaxTxHash  bool
	StateRootProvider  StateRootProvider
}

func (t *OGTxCreator) GetArchiveNonce() uint64 {
	return atomic.AddUint64(&t.archiveNonce, 1)
}

func (t *OGTxCreator) Stop() {
	t.quit = true
}

func (m *OGTxCreator) newUnsignedTx(req UnsignedTxBuildRequest) types.Txi {
	tx := tx_types.Tx{
		Value:   req.Value,
		To:      req.To,
		From:    &req.From,
		TokenId: req.TokenId,
		TxBase: types.TxBase{
			AccountNonce: req.AccountNonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	return &tx
}

func (m *OGTxCreator) NewArchiveWithSeal(data []byte) (tx types.Txi, err error) {
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

func (m *OGTxCreator) NewTxWithSeal(req TxWithSealBuildRequest) (tx types.Txi, err error) {
	tx = &tx_types.Tx{
		From: &req.From,
		// TODO
		// should consider the case that to is nil. (contract creation)
		To:      req.To,
		Value:   req.Value,
		Data:    req.Data,
		TokenId: req.TokenId,
		TxBase: types.TxBase{
			AccountNonce: req.Nonce,
			Type:         types.TxBaseTypeNormal,
		},
	}
	tx.GetBase().Signature = req.Sig.Bytes
	tx.GetBase().PublicKey = req.Pubkey.Bytes

	if ok := m.SealTx(tx, nil); !ok {
		logrus.Warn("failed to seal tx")
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")

	return tx, nil
}

func (m *OGTxCreator) NewActionTxWithSeal(req ActionTxBuildRequest) (tx types.Txi, err error) {
	tx = &tx_types.ActionTx{
		From: &req.From,
		// TODO
		// should consider the case that to is nil. (contract creation)
		TxBase: types.TxBase{
			AccountNonce: req.AccountNonce,
			Type:         types.TxBaseAction,
		},
		Action: req.Action,
		ActionData: &tx_types.PublicOffering{
			Value:     req.Value,
			EnableSPO: req.EnableSpo,
			TokenId:   req.TokenId,
			TokenName: req.TokenName,
		},
	}
	tx.GetBase().Signature = req.Sig.Bytes
	tx.GetBase().PublicKey = req.Pubkey.Bytes

	if ok := m.SealTx(tx, nil); !ok {
		err = fmt.Errorf("failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	tx.SetVerified(types.VerifiedFormat)
	return tx, nil
}

func (m *OGTxCreator) NewSignedTx(req SignedTxBuildRequest) types.Txi {
	if req.PrivateKey.Type != crypto.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.newUnsignedTx(req.UnsignedTxBuildRequest)
	// do sign work
	signature := crypto.Signer.Sign(req.PrivateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = crypto.Signer.PubKey(req.PrivateKey).Bytes
	return tx
}

func (m *OGTxCreator) newUnsignedSequencer(req UnsignedSequencerBuildRequest) *tx_types.Sequencer {
	tx := tx_types.Sequencer{
		Issuer: &req.Issuer,
		TxBase: types.TxBase{
			AccountNonce: req.AccountNonce,
			Type:         types.TxBaseTypeSequencer,
			Height:       req.Height,
		},
	}
	return &tx
}

//NewSignedSequencer this function is for test
func (m *OGTxCreator) NewSignedSequencer(req SignedSequencerBuildRequest) types.Txi {
	if req.PrivateKey.Type != crypto.Signer.GetCryptoType() {
		panic("crypto type mismatch")
	}
	tx := m.newUnsignedSequencer(req.UnsignedSequencerBuildRequest)
	// do sign work
	logrus.Tracef("seq before sign, the sign type is: %s", crypto.Signer.GetCryptoType().String())
	signature := crypto.Signer.Sign(req.PrivateKey, tx.SignatureTargets())
	tx.GetBase().Signature = signature.Bytes
	tx.GetBase().PublicKey = crypto.Signer.PubKey(req.PrivateKey).Bytes
	return tx
}

// validateGraphStructure validates if parents are not conflicted, not double spending or other misbehaviors
func (m *OGTxCreator) validateGraphStructure(parents []types.Txi) (ok bool) {
	ok = true
	for _, parent := range parents {
		ok = ok && m.GraphVerifier.Verify(parent)
		if !ok {
			return
		}
	}
	return
}

func (m *OGTxCreator) tryConnect(tx types.Txi, parents []types.Txi, privateKey *crypto.PrivateKey) (txRet types.Txi, ok bool) {
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
			logrus.WithField("tx ", tx).Debug("NOT OK")
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
func (m *OGTxCreator) SealTx(tx types.Txi, priveKey *crypto.PrivateKey) (ok bool) {
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
			// Actually, this value is already set during mining.
			// Incase that other implementation does not do that, re-assign
			tx.GetBase().MineNonce = minedNonce
			//logrus.Debugf("Total time for Mining: %d ns, %d times", time.Since(timeStart).Nanoseconds(), minedNonce)
			// pick up parents.
			for i := 0; i < m.MaxConnectingTries; i++ {
				if m.quit {
					logrus.Info("got quit signal")
					return false
				}
				connectionTries++
				var txs types.Txis
				var ancestor types.Txi
				if tx.GetType() != types.TxBaseTypeArchive {
					ancestor = m.TipGenerator.GetByNonce(tx.Sender(), tx.GetNonce()-1)
				}
				// if there is a previous my tx that is in current seq,
				// use it as my parent.
				// it is required to accelerate validating
				if ancestor != nil && !ancestor.InValid() {
					txs = m.TipGenerator.GetRandomTips(2)
					var include bool
					for _, tx := range txs {
						if tx.GetTxHash() == ancestor.GetTxHash() {
							include = true
							break
						}
					}
					if !include && len(txs) > 0 {
						txs[0] = ancestor
					}

				} else {
					txs = m.TipGenerator.GetRandomTips(2)
				}

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
				} else {
					logrus.WithField("parents ", txs).WithField("connection tries ", connectionTries).WithField("tx ", tx).Debug("NOT OK")
				}
			}
			if mineCount > 1 {
				return false
			}
		case <-time.NewTimer(time.Minute * 5).C:
			m.Miner.Stop()
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

func (m *OGTxCreator) GenerateSequencer(issuer common.Address, height uint64, accountNonce uint64,
	privateKey *crypto.PrivateKey, blsPubKey []byte) (seq *tx_types.Sequencer, reterr error, genAgain bool) {

	tx := m.newUnsignedSequencer(UnsignedSequencerBuildRequest{
		Height:       height,
		Issuer:       issuer,
		AccountNonce: accountNonce,
	})
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
			reterr = errors.New("quit")
			return nil, reterr, false
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
			if reterr = m.TipGenerator.IsBadSeq(tx); reterr != nil {
				return nil, reterr, true
			}
			continue
		} else {
			//calculate root
			//calculate signatrue
			root, err := m.StateRootProvider.PreConfirm(tx)
			if err != nil {
				logrus.WithField("seq ", tx).Errorf("CalculateStateRoot err  %v", err)
				return nil, err, false
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
		return tx, nil, false
	}
	logrus.WithFields(logrus.Fields{
		"elapsedns":  time.Since(timeStart).Nanoseconds(),
		"re-connect": connectionTries,
	}).Warnf("generate sequencer failed")
	return nil, nil, false
}

func (t *OGTxCreator) ValidateSequencer(seq tx_types.Sequencer) error {
	// TODO: validate sequencer's graph structure and txs being confirmed.
	// using Preconfirm in tx_pool

}
