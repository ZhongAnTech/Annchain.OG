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
package syncer

import (
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/types"

	"github.com/annchain/OG/protocol"
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
)

var MaxBufferSiza = 4096 * 16

type SyncBuffer struct {
	Txs        map[common.Hash]types.Txi
	TxsList    common.Hashes
	Seq        *types.Sequencer
	mu         sync.RWMutex
	txPool     og.ITxPool
	dag        og.IDag
	acceptTxs  uint32
	quitHandel bool
	Verifiers  []protocol.Verifier
}

type SyncBufferConfig struct {
	TxPool    og.ITxPool
	Verifiers []protocol.Verifier
	Dag       og.IDag
}

func DefaultSyncBufferConfig(txPool og.ITxPool, dag og.IDag, Verifiers []protocol.Verifier) SyncBufferConfig {
	config := SyncBufferConfig{
		TxPool:    txPool,
		Dag:       dag,
		Verifiers: Verifiers,
	}
	return config
}

func (s *SyncBuffer) Name() string {
	return "SyncBuffer"
}

func NewSyncBuffer(config SyncBufferConfig) *SyncBuffer {
	s := &SyncBuffer{
		Txs:       make(map[common.Hash]types.Txi),
		txPool:    config.TxPool,
		dag:       config.Dag,
		Verifiers: config.Verifiers,
	}
	return s
}

func (s *SyncBuffer) Start() {
	log.Info("Syncbuffer started")
}

func (s *SyncBuffer) Stop() {
	log.Info("Syncbuffer will stop")
	s.quitHandel = true
}

// range map is random value ,so store hashs using slice
func (s *SyncBuffer) addTxs(txs types.Txis, seq *types.Sequencer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Seq = seq
	for _, tx := range txs {
		if len(s.Txs) > MaxBufferSiza {
			return fmt.Errorf("too much txs")
		}
		if tx == nil {
			log.Debug("nil tx")
			continue
		}
		if _, ok := s.Txs[tx.GetTxHash()]; !ok {

			s.Txs[tx.GetTxHash()] = tx
		}
		s.TxsList = append(s.TxsList, tx.GetTxHash())
	}
	return nil

}

func (s *SyncBuffer) AddTxs(seq *types.Sequencer, txs types.Txis) error {
	if atomic.LoadUint32(&s.acceptTxs) == 0 {
		atomic.StoreUint32(&s.acceptTxs, 1)
		defer atomic.StoreUint32(&s.acceptTxs, 0)
		s.clean()
		if seq == nil {
			err := fmt.Errorf("nil sequencer")
			log.WithError(err).Debug("add txs error")
			return err
		}
		if seq.Height != s.dag.LatestSequencer().Height+1 {
			log.WithField("latests seq height ", s.dag.LatestSequencer().Height).WithField(
				"seq height", seq.Height).Warn("id mismatch")
			return nil
		}
		err := s.addTxs(txs, seq)
		if err != nil {
			log.WithError(err).Debug("add txs error")
			return err
		}
		err = s.Handle()
		if err != nil {
			logrus.WithError(err).WithField("txs ", txs).WithField("seq ", seq).Warn("handle tx error")
		}
		return err

	} else {
		err := fmt.Errorf("addtx busy")
		return err
	}
}

func (s *SyncBuffer) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Txs)
}

func (s *SyncBuffer) Get(hash common.Hash) types.Txi {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Txs[hash]

}

func (s *SyncBuffer) GetAllKeys() common.Hashes {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys common.Hashes
	for k := range s.Txs {
		keys = append(keys, k)
	}
	return keys
}

func (s *SyncBuffer) clean() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.Txs {
		delete(s.Txs, k)
	}
	s.TxsList = nil
}

func (s *SyncBuffer) Handle() error {
	if s.quitHandel {
		return nil
	}
	count := s.Count()
	log.WithField("txs len", count).WithField("seq", s.Seq).Trace("handle txs start")
	err := s.verifyElders(s.Seq)
	if err != nil {
		log.WithField("txlist ", s.TxsList).WithField("seq ", s.Seq).WithError(err).Warn("handle fail")
		return err
	}
	log.WithField("len ", len(s.Verifiers)).Debug("len sync buffer verifier")
	for _, hash := range s.TxsList {
		tx := s.Get(hash)
		if tx == nil {
			panic("never come here")
		}
		//if tx is already in txbool ,no need to verify again
		if s.txPool.IsLocalHash(hash) {
			continue
		}
		for _, verifier := range s.Verifiers {
			if !verifier.Verify(tx) {
				log.WithField("tx", tx).Warn("bad tx")
				err = errors.New("bad tx format")
				goto out
			}
		}
		//need to feedback to buffer
		err = s.txPool.AddRemoteTx(tx, false)
		if err != nil {
			//this transaction received by broadcast ,so don't return err
			if err == types.ErrDuplicateTx {
				err = nil
				continue
			} else {
				goto out
			}
		}
	}

out:
	if err == nil {
		log.WithField("id", s.Seq).Trace("before add seq")
		for _, verifier := range s.Verifiers {
			if !verifier.Verify(s.Seq) {
				log.WithField("tx", s.Seq).Warn("bad seq")
				err = errors.New("bad seq format")
			}
		}
		if err == nil {
			err = s.txPool.AddRemoteTx(s.Seq, false)
			log.WithField("id", s.Seq).Trace("after add seq")
		}
	}
	if err != nil {
		log.WithField("seq ", s.Seq).WithError(err).Warn("handel fail")
		//panic("handle fail for test")
	} else {
		log.WithField("txs len", count).WithField("seq ", s.Seq).Debug("handle txs done")

	}
	return err
}

func (s *SyncBuffer) verifyElders(seq types.Txi) error {

	allKeys := s.GetAllKeys()
	keysMap := make(map[common.Hash]int)
	for _, k := range allKeys {
		keysMap[k] = 1
	}

	inSeekingPool := map[common.Hash]int{}
	seekingPool := common.Hashes{}
	for _, parentHash := range seq.Parents() {
		seekingPool = append(seekingPool, parentHash)
		// seekingPool.PushBack(parentHash)
	}
	for len(seekingPool) > 0 {
		elderHash := seekingPool[0]
		seekingPool = seekingPool[1:]
		// elderHash := seekingPool.Remove(seekingPool.Front()).(common.Hash)

		elder := s.Get(elderHash)
		if elder == nil {
			if s.txPool.IsLocalHash(elderHash) {
				continue
			}
			err := fmt.Errorf("parent not found")
			log.WithField("hash", elderHash.String()).Warn("parent not found")
			return err
		}
		delete(keysMap, elderHash)
		for _, elderParentHash := range elder.Parents() {
			if _, in := inSeekingPool[elderParentHash]; !in {
				seekingPool = append(seekingPool, elderParentHash)
				// seekingPool.PushBack(elderParentHash)
				inSeekingPool[elderParentHash] = 0
			}
		}
	}
	if len(keysMap) != 0 {
		err := fmt.Errorf("txs number mismatch,redundent txs  %d", len(allKeys))
		return err
	}
	return nil
}
