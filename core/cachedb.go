package core

import (
	"fmt"
	"sync"
	"time"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/common/math"
	log "github.com/sirupsen/logrus"
)

type CacheDBConfig struct {
	FlushTimer		time.Duration
	PurgeTimer		time.Duration
	BeatExpireTime	time.Duration
}
func DefaultCacheDBConfig() CacheDBConfig {
	return CacheDBConfig{
		FlushTimer: time.Duration(5000),
		PurgeTimer: time.Duration(10000),
		BeatExpireTime: time.Duration(10 * time.Minute),
	}
}

type CacheDB struct {
	conf		CacheDBConfig

	db			ogdb.Database
	accessor	*Accessor

	txs			*CacheTxs
	states		map[types.Address]*State
	dirtyset	map[types.Address]struct{}
	beats		map[types.Address]time.Time

	close		chan struct{}

	mu 			sync.RWMutex
}
func NewCacheDB(conf CacheDBConfig, db ogdb.Database, acc *Accessor) *CacheDB {
	cd := &CacheDB{
		conf:		conf,
		db:			db,
		accessor:	acc,
		txs:		NewCacheTxs(),
		states:		make(map[types.Address]*State),
		dirtyset:	make(map[types.Address]struct{}),
		beats:		make(map[types.Address]time.Time),
		close:		make(chan struct{}),
	}

	go cd.loop()
	return cd
}

func (cd *CacheDB) Stop() {
	close(cd.close)
}

// CreateNewState will create a new state for input address and 
// return the state. If input address already exists in CacheDB 
// it returns an error.
func (cd *CacheDB) CreateNewState(addr types.Address) (*State, error) {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	state := cd.states[addr]
	if state != nil {
		return nil, fmt.Errorf("state already exists, addr: %s", addr.String())
	}
	
	state = NewState(addr)
	cd.states[addr] = state
	cd.beats[addr] = time.Now()

	return state, nil
}

func (cd *CacheDB) GetBalance(addr types.Address) *math.BigInt {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	return cd.getBalance(addr)
}
func (cd *CacheDB) getBalance(addr types.Address) *math.BigInt {
	state, err := cd.getState(addr)
	if err != nil {
		return math.NewBigInt(0)
	}
	return state.Balance
}

func (cd *CacheDB) GetNonce(addr types.Address) (uint64, error) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	return cd.getNonce(addr)
}
func (cd *CacheDB) getNonce(addr types.Address) (uint64, error) {
	state, err := cd.getState(addr)
	if err != nil {
		return 0, types.ErrNonceNotExist
	}
	return state.Nonce, nil
}

// GetState get a state from CacheDB. If state not exist, 
// load it from db. 
func (cd *CacheDB) GetState(addr types.Address) (*State, error) {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	return cd.getState(addr)
}
func (cd *CacheDB) getState(addr types.Address) (*State, error) {
	state, exist := cd.states[addr]
	if !exist {
		var err error
		state, err = cd.loadState(addr)
		if err != nil {
			return nil, err
		}
		cd.updateState(addr, state)
	}
	return state, nil
}

// GetOrCreateState
func (cd *CacheDB) GetOrCreateState(addr types.Address) *State {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	return cd.getOrCreateState(addr)
}
func (cd *CacheDB) getOrCreateState(addr types.Address) *State {
	state, _ := cd.getState(addr)
	if state == nil {
		state = NewState(addr)
	}
	cd.updateState(addr, state)
	return state
}

// DeleteState remove a state from CacheDB. Return error 
// if it fails.
func (cd *CacheDB) DeleteState(addr types.Address) error {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	return cd.deleteState(addr)
}
func (cd *CacheDB) deleteState(addr types.Address) error {
	_, exist := cd.states[addr]
	if !exist {
		return nil
	}
	delete(cd.states, addr)
	return nil
}

// UpdateState set addr's state in CacheDB. 
// 
// Note that this setting will force updating the CacheDB without 
// any verification. Call this UpdateState carefully.
func (cd *CacheDB) UpdateState(addr types.Address, state *State) {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.updateState(addr, state)
}
func (cd *CacheDB) updateState(addr types.Address, state *State) {
	cd.states[addr] = state
	cd.dirtyset[addr] = struct{}{}
	cd.refreshbeat(addr)
}

func (cd *CacheDB) AddBalance(addr types.Address, increment *math.BigInt) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	
	state := cd.getOrCreateState(addr)
	state.AddBalance(increment)
	
	cd.updateState(addr, state)
}

func (cd *CacheDB) SubBalance(addr types.Address, decrement *math.BigInt) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	
	state := cd.getOrCreateState(addr)
	state.SubBalance(decrement)
	
	cd.updateState(addr, state)
}

func (cd *CacheDB) SetBalance(addr types.Address, balance *math.BigInt) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	
	state := cd.getOrCreateState(addr)
	state.SetBalance(balance)
	
	cd.updateState(addr, state)
}

func (cd *CacheDB) SetNonce(addr types.Address, nonce uint64) {
	cd.mu.Lock()
	defer cd.mu.Unlock()
	
	state := cd.getOrCreateState(addr)
	state.SetNonce(nonce)
	
	cd.updateState(addr, state)
}

// Load data from db.
func (cd *CacheDB) loadState(addr types.Address) (*State, error) {
	return cd.accessor.LoadState(addr)
}

func (cd *CacheDB) GetTxByHash(hash types.Hash) types.Txi {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	tx := cd.txs.GetTxByHash(hash)
	if tx == nil {
		// TODO load data from db
	}

	return tx
}

func (cd *CacheDB) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	return cd.txs.GetTxByNonce(addr, nonce)
}

func (cd *CacheDB) WriteTransaction(tx types.Txi) error {

	return nil
}

func (cd *CacheDB) RemoveTxByHash(hash types.Hash) error {

	return nil
}

func (cd *CacheDB) RemoveTxByNonce(addr types.Address, nonce uint64) types.Txi {

	return nil
}

// flush tries to save those dirty data to disk db.
func (cd *CacheDB) flush() {
	for addr, _ := range cd.dirtyset {
		state, exist := cd.states[addr]
		if !exist {
			log.Warnf("can't find dirty state in CacheDB, addr: %s", addr.String())
			continue
		}
		cd.accessor.SaveState(addr, state)
	}
	cd.dirtyset = make(map[types.Address]struct{})
}

// purge tries to remove all the state that haven't sent any beat 
// for a long time. 
// 
// Note that dirty states will not be removed.
func (cd *CacheDB) purge() {
	for addr, lastbeat := range cd.beats {
		// skip dirty states
		if _, in := cd.dirtyset[addr]; in {
			continue
		}
		if time.Since(lastbeat) > cd.conf.BeatExpireTime {
			cd.deleteState(addr)
		}
	}
}

// refreshbeat update the beat time of an address.
func (cd *CacheDB) refreshbeat(addr types.Address) {
	state := cd.states[addr]
	if state == nil {
		return
	}
	cd.beats[addr] = time.Now()
}

func (cd *CacheDB) loop() {
	flushTimer := time.NewTicker(cd.conf.FlushTimer)
	purgeTimer := time.NewTicker(cd.conf.PurgeTimer)

	for {
		select {
		case <-cd.close:
			flushTimer.Stop()
			purgeTimer.Stop()
			cd.mu.Lock()
			cd.flush()
			cd.mu.Unlock()
			return

		case <-flushTimer.C:
			cd.mu.Lock()
			cd.flush()
			cd.mu.Unlock()

		case <-purgeTimer.C:
			cd.mu.Lock()
			cd.purge()
			cd.mu.Unlock()

		}
	}
}


type CacheTxs struct {
	txmap *TxMap
	txlist	map[types.Address]*TxList
}
func NewCacheTxs() *CacheTxs {
	return &CacheTxs{
		txmap: NewTxMap(),
		txlist: make(map[types.Address]*TxList),
	}
}

func (ctxs *CacheTxs) GetTxByHash(hash types.Hash) types.Txi {
	return ctxs.txmap.Get(hash)
}

func (ctxs *CacheTxs) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
	txlist, ok := ctxs.txlist[addr]
	if !ok {
		return nil
	}
	return txlist.Get(nonce)
}

func (ctxs *CacheTxs) AddTx(tx types.Txi) {
	txlist, ok := ctxs.txlist[tx.Sender()]
	if !ok {
		txlist = NewTxList()
		ctxs.txlist[tx.Sender()] = txlist
	}
	txlist.Put(tx)
	ctxs.txmap.Add(tx)
}

func (ctxs *CacheTxs) RemoveTxByHash(hash types.Hash) error {
	tx := ctxs.txmap.Get(hash)
	if tx == nil {
		return nil
	}
	ctxs.txmap.Remove(hash)

	txlist, ok := ctxs.txlist[tx.Sender()]
	if !ok {
		return fmt.Errorf("tx not exist in txlist.")
	}
	txlist.Remove(tx.GetNonce())
	
	return nil
}

func (ctxs *CacheTxs) RemoveTxByNonce(addr types.Address, nonce uint64) error {
	txlist, ok := ctxs.txlist[addr]
	if !ok {
		return nil
	}
	tx := txlist.Get(nonce)
	if tx == nil {
		return nil
	}
	ctxs.txmap.Remove(tx.GetTxHash())
	txlist.Remove(nonce)
	return nil
}
