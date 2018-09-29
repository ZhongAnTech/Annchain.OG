package og

import (
	"container/list"
	"fmt"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type SyncBuffer struct {
	Txs        map[types.Hash]types.Txi
	Seq        *types.Sequencer
	mu         sync.RWMutex
	txPool     ITxPool
	txBuffer   ITxBuffer
	acceptTxs  uint32
	quitHandel bool
}

type SyncBufferConfig struct {
	TxPool   ITxPool
	TxBuffer ITxBuffer
}

func (s *SyncBuffer) Start() {
	log.Info("Syncbuffer started")
}

func (s *SyncBuffer) Stop() {
	log.Info("Syncbuffer will stop")
	s.quitHandel = true
}

// range map is random value ,so store hashs using slice
func (s *SyncBuffer) addTxs(txs []types.Txi, seq *types.Sequencer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq == nil {
		err := fmt.Errorf("nil sequencer")
		log.WithError(err).Debug("add txs error")
		return err
	}
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
	}
	return nil

}

func (s *SyncBuffer) AddTxs(txs []types.Txi, seq *types.Sequencer) error {
	if atomic.LoadUint32(&s.acceptTxs) == 0 {
		atomic.StoreUint32(&s.acceptTxs, 1)
		defer atomic.StoreUint32(&s.acceptTxs, 0)
		s.clean()
		err := s.addTxs(txs, seq)
		if err != nil {
			return err
		}
		s.Handle()

	} else {
		err := fmt.Errorf("addtx busy")
		return err
	}
	return nil
}

func (s *SyncBuffer) Name() string {
	return "SyncBuffer"
}

var MaxBufferSiza = 4096 * 4

func NewSyncBuffer(config SyncBufferConfig) *SyncBuffer {
	s := &SyncBuffer{
		Txs:      make(map[types.Hash]types.Txi),
		txPool:   config.TxPool,
		txBuffer: config.TxBuffer,
	}
	return s
}

func (s *SyncBuffer) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Txs)
}

func (s *SyncBuffer) Get(hash types.Hash) types.Txi {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Txs[hash]

}

func (s *SyncBuffer) GetAllKeys() []types.Hash {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var keys []types.Hash
	for k, _ := range s.Txs {
		keys = append(keys, k)
	}
	return keys
}

func (s *SyncBuffer) Remove(hash types.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Txs, hash)
}

func (s *SyncBuffer) clean() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, _ := range s.Txs {
		delete(s.Txs, k)
	}
}

func (s *SyncBuffer) Handle() error {
	if s.quitHandel {
		return nil
	}
	count := s.Count()
	log.WithField("txs len", count).WithField("seq id ", s.Seq.Number()).Debug("handle txs start")
	err := s.verifyElders(s.Seq)
	if err != nil {
		log.WithField("seq ", s.Seq.Number()).WithError(err).Warn("handel fail")
		return err
	}

	for _, tx := range s.Txs {
		// temporary commit for testing
		//todo
		/*
			     	err =  s.txBuffer.verifyTxFormat(tx)
			     	if err!=nil {
						break
					}
		*/
		err = s.txPool.AddRemoteTx(tx)
		if err != nil {
			break
		}
	}
	if err == nil {
		log.WithField("id", s.Seq.Number()).Debug("before add seq")
		err = s.txPool.AddRemoteTx(s.Seq)
		log.WithField("id", s.Seq.Number()).Debug("after add seq")
	}
	if err != nil {
		log.WithField("seq ", s.Seq.Number()).WithError(err).Warn("handel fail")
	} else {
		log.WithField("txs len", count).WithField("seq id ", s.Seq.Number()).Debug("handle txs done")

	}
	return err
}

func (s *SyncBuffer) verifyElders(seq types.Txi) error {

	allKeys := s.GetAllKeys()
	keysMap := make(map[types.Hash]int)
	for _, k := range allKeys {
		keysMap[k] = 1
	}

	inSeekingPool := map[types.Hash]int{}
	seekingPool := list.New()
	for _, parentHash := range seq.Parents() {
		seekingPool.PushBack(parentHash)
	}
	for seekingPool.Len() > 0 {
		elderHash := seekingPool.Remove(seekingPool.Front()).(types.Hash)
		elder := s.Get(elderHash)
		if elder == nil {
			if s.txBuffer.isLocalHash(elderHash) {
				continue
			}
			err := fmt.Errorf("parent not found ")
			log.WithField("hash", elderHash.String()).Warn("elser not found")
			return err
		}
		delete(keysMap, elderHash)
		for _, elderParentHash := range elder.Parents() {
			if _, in := inSeekingPool[elderParentHash]; !in {
				seekingPool.PushBack(elderParentHash)
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
