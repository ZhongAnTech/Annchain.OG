package og

import (
	"fmt"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
	"time"
)

type SyncBuffer struct {
	Txs       map[types.Hash]types.Txi
	mu        sync.RWMutex
	txBuffer  *TxBuffer
	acceptTxs uint32
	quit      chan bool
	start     chan bool
	done      chan bool
}

func (s *SyncBuffer) Start() {
	go s.loop()
}

func (s *SyncBuffer) Stop() {
	s.quit <- true
}

func (s *SyncBuffer) addTxs(txs []types.Txi) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tx := range txs {
		if len(s.Txs) > MaxBufferSiza {
			return fmt.Errorf("too much txs")
		}
		if _, ok := s.Txs[tx.GetTxHash()]; !ok {
			s.Txs[tx.GetTxHash()] = tx
		}
	}
	return nil

}

func (s *SyncBuffer) AddTxs(txs []types.Txi) error {
	if atomic.LoadUint32(&s.acceptTxs) == 0 {
		s.addTxs(txs)
		s.start <- true
	} else {
		for {
			select {
			case <-s.done:
				s.addTxs(txs)
				s.start <- true
				return nil
			case <-time.After(time.Millisecond * 100):
				if atomic.LoadUint32(&s.acceptTxs) == 0 {
					s.addTxs(txs)
					s.start <- true
					return nil
				}
			}
		}
	}
	return nil
}

func (s *SyncBuffer) Name() string {
	return "TxBuffer"
}

func (s *SyncBuffer) loop() {
	for {
		select {
		case <-s.quit:
			log.Info("TxBuffer received quit message. Quitting...")
			return
		case <-s.start:
			atomic.StoreUint32(&s.acceptTxs, 1)
			s.Handle()
			atomic.StoreUint32(&s.acceptTxs, 0)
			s.done <- true
		}
	}
}

var MaxBufferSiza = 4096 * 4

func NewSyncBuffer(buffer *TxBuffer) *SyncBuffer {

	s := &SyncBuffer{
		Txs:   make(map[types.Hash]types.Txi),
		quit:  make(chan bool),
		start: make(chan bool),
		done:  make(chan bool),
	}
	s.txBuffer = buffer
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
	// slice of keys
	for k := range s.Txs {
		keys = append(keys, k)
	}
	return keys
}

func (s *SyncBuffer) GetAllValues() []types.Txi {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var values []types.Txi
	// slice of keys
	for _, v := range s.Txs {
		values = append(values, v)
	}
	return values
}

func (s *SyncBuffer) Exists(tx types.Txi) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.Txs[tx.GetTxHash()]; !ok {
		return false
	}
	return true
}

func (s *SyncBuffer) Remove(hash types.Hash) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.Txs, hash)
}

func (s *SyncBuffer) Add(tx types.Txi) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Txs) > MaxBufferSiza {
		return fmt.Errorf("too much txs")
	}
	if _, ok := s.Txs[tx.GetTxHash()]; !ok {
		s.Txs[tx.GetTxHash()] = tx
	}
	return nil
}

func (s *SyncBuffer) Handle() {

	txHashs := s.GetAllKeys()
	for _, txHash := range txHashs {
		s.HandelOne(txHash)
	}
	if s.Count() == 0 {
		log.Info("finished processing txs")
	}
	return
}

func (s *SyncBuffer) HandelOne(hash types.Hash) (added bool, err error) {
	b := s.txBuffer
	tx := s.Get(hash)
	if tx == nil {
		s.Remove(tx.GetTxHash())
		return false, nil
	}
	// already in the dag or tx_pool.
	if b.isKnownHash(tx.GetTxHash()) {
		s.Remove(tx.GetTxHash())
		return true, nil
	}
	log.Debug("hande sync tx ", tx.GetTxHash())

	//if parent is in dag or pool , verify and add tx to pool
	//else if parent is in sync_buffer ,process parent first
	//else if parent not found , got it
	//
	var unkown bool
	for _, pHash := range tx.Parents() {
		if !b.isKnownHash(pHash) {
			unkown = true
			parent := s.Get(pHash)
			if parent == nil {
				log.WithField("hash", tx.GetTxHash()).Warn("miss parents,drop this tx")
				s.Remove(tx.GetTxHash())
				return false, fmt.Errorf("parent not found")
			} else {
				if result, _ := s.HandelOne(parent.GetTxHash()); result {
					unkown = false
				}
			}
		}
	}
	if !unkown {
		if err := b.verifyTxFormat(tx); err != nil {
			log.WithError(err).Debugf("Received invalid tx %s", tx.GetTxHash().Hex())
			s.Remove(tx.GetTxHash())
			return false, fmt.Errorf("invalid txs")
		}
		s.Remove(tx.GetTxHash())
		s.txBuffer.txPool.AddRemoteTx(tx)
		return true, nil
	}

	return false, nil
}
