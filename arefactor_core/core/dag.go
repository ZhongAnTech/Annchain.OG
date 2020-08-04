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
package core

import (
	"fmt"
	"sort"
	"sync"

	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/OG/arefactor_core/core/state"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	evm "github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	vmtypes "github.com/annchain/OG/vm/types"

	log "github.com/sirupsen/logrus"
)

var (
	// empty address is the address used for contract creation, it
	// is filled in [tx.To].
	emptyAddress = ogTypes.BytesToAddress20(nil)

	emptyRoot = ogTypes.BytesToHash32(nil)

	DefaultGasLimit = uint64(10000000000)

	DefaultCoinbase, _ = ogTypes.HexToAddress20("0x1234567812345678AABBCCDDEEFF998877665544")
)

type DagConfig struct {
	GenesisPath string
}

//type Ledger interface {
//	GetTx(hash ogTypes.Hash) types.Txi
//	GetBalance(addr ogTypes.Address, tokenID int32) *math.BigInt
//	GetLatestNonce(addr ogTypes.Address) (uint64, error)
//}

type Dag struct {
	conf DagConfig

	db       ogdb.Database
	accessor *Accessor

	txProcessor TxProcessor
	vmProcessor VmProcessor

	genesis            *types.Sequencer
	latestSequencer    *types.Sequencer
	latestStateDB      *state.StateDB
	confirmedSequencer *types.Sequencer
	confirmedStateDB   *state.StateDB
	cachedBatches      *CachedConfirms // stores speculated sequencers, use state root as batch key
	pendedBatches      *CachedConfirms // stores pushed sequencers, use sequencer hash as batch key

	//txcached *txcached

	OnConsensusTXConfirmed chan []types.Txi
	close                  chan struct{}

	mu sync.RWMutex
}

func NewDag(conf DagConfig, stateDBConfig state.StateDBConfig, db ogdb.Database, txProcessor TxProcessor, vmProcessor VmProcessor) (*Dag, error) {
	dag := &Dag{
		conf:     conf,
		db:       db,
		accessor: NewAccessor(db),
		//txcached:    newTxcached(10000), // TODO delete txcached later
		txProcessor:   txProcessor,
		vmProcessor:   vmProcessor,
		cachedBatches: newCachedConfirms(),
		pendedBatches: newCachedConfirms(),
		close:         make(chan struct{}),
	}

	genesis, latestConfirmedSeq := dag.LoadLatestConfirmedSeq()
	if genesis == nil {
		genesis, balance := DefaultGenesis(conf.GenesisPath)
		statedb, err := state.NewStateDB(stateDBConfig, state.NewDatabase(db), emptyRoot)
		if err != nil {
			return nil, fmt.Errorf("create statedb err: %v", err)
		}
		dag.latestStateDB = statedb
		dag.confirmedStateDB = statedb
		if err := dag.Init(genesis, balance); err != nil {
			return nil, err
		}
		return dag, nil
	}
	dag.genesis = genesis
	dag.latestSequencer = latestConfirmedSeq
	dag.confirmedSequencer = latestConfirmedSeq

	root := latestConfirmedSeq.StateRoot
	log.Infof("the root loaded from last state is: %x", root.Bytes())
	statedb, err := state.NewStateDB(stateDBConfig, state.NewDatabase(db), root)
	if err != nil {
		return nil, fmt.Errorf("create statedb err: %v", err)
	}
	dag.latestStateDB = statedb
	dag.confirmedStateDB = statedb

	if dag.GetHeight() > 0 && root.Length() == 0 {
		panic("should not be empty hash. Database may be corrupted. Please clean datadir")
	}
	return dag, nil
}

func DefaultDagConfig() DagConfig {
	return DagConfig{}
}

func (dag *Dag) Start() {
	log.Infof("Dag Start")

}

func (dag *Dag) Stop() {
	close(dag.close)

	dag.latestStateDB.Stop()
	for _, batch := range dag.pendedBatches.batches {
		batch.stop()
	}
	log.Infof("Dag Stopped")
}

// Init inits genesis sequencer and genesis state of the network.
func (dag *Dag) Init(genesis *types.Sequencer, genesisBalance map[ogTypes.AddressKey]*math.BigInt) error {
	if genesis.Height != 0 {
		return fmt.Errorf("invalheight genesis: height is not zero")
	}
	var err error

	// init genesis balance
	for addrKey, value := range genesisBalance {
		addr, _ := ogTypes.HexToAddress20(string(addrKey))
		dag.latestStateDB.SetBalance(addr, value)
	}
	// commit state and init a genesis state root
	root, err := dag.latestStateDB.Commit()
	if err != nil {
		return fmt.Errorf("commit genesis state err: %v", err)
	}
	trieDB := dag.latestStateDB.Database().TrieDB()
	err = trieDB.Commit(root, false)
	if err != nil {
		return fmt.Errorf("commit genesis trie err: %v", err)
	}

	// write genesis
	genesis.StateRoot = root
	err = dag.accessor.WriteGenesis(genesis)
	if err != nil {
		return err
	}
	err = dag.writeSequencer(nil, genesis)
	if err != nil {
		return err
	}

	// set latest sequencer
	err = dag.accessor.WriteLatestConfirmedSeq(nil, genesis)
	if err != nil {
		return err
	}
	log.Tracef("successfully store genesis: %s", genesis)

	dag.genesis = genesis
	dag.latestSequencer = genesis
	dag.confirmedSequencer = genesis

	log.Infof("Dag finish init")
	return nil
}

// LoadLatestConfirmedSeq load genesis and latest confirmed sequencer from db.
func (dag *Dag) LoadLatestConfirmedSeq() (*types.Sequencer, *types.Sequencer) {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	genesis := dag.accessor.ReadGenesis()
	if genesis == nil {
		return nil, nil
	}
	dag.genesis = genesis

	seq := dag.accessor.ReadLatestConfirmedSeq()
	if seq == nil {
		return nil, nil
	} else {
		return genesis, seq
	}
}

// Genesis returns the genesis tx of dag
func (dag *Dag) Genesis() *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.genesis
}

// LatestSequencer returns the latest sequencer stored in dag
func (dag *Dag) LatestSequencer() *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.latestSequencer
}

func (dag *Dag) LatestSequencerHash() ogTypes.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.latestSequencer.GetTxHash()
}

func (dag *Dag) ConfirmedSequencer() *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.confirmedSequencer
}

func (dag *Dag) ConfirmedSequencerHash() ogTypes.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.confirmedSequencer.GetTxHash()
}

func (dag *Dag) LatestConfirmedSequencer() *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.confirmedSequencer
}

//GetHeight get cuurent height
func (dag *Dag) GetHeight() uint64 {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.latestSequencer.Height
}

func (dag *Dag) GetConfirmedHeight() uint64 {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.confirmedSequencer.Height

}

//// Accessor returns the db accessor of dag
//func (dag *Dag) Accessor() *Accessor {
//	return dag.accessor
//}

// Push trys to move a tx from tx pool to dag db.
func (dag *Dag) Push(batch *PushBatch) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	return dag.push(batch)
}

// GetTx gets tx from dag network indexed by tx hash
func (dag *Dag) GetTx(hash ogTypes.Hash) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTx(hash)
}

func (dag *Dag) getTx(hash ogTypes.Hash) types.Txi {
	tx, _ := dag.pendedBatches.getTxAndReceipt(hash)
	if tx != nil {
		return tx
	}
	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) getConfirmedTx(hash ogTypes.Hash) types.Txi {
	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) ExistTx(baseSeqHash ogTypes.Hash, hash ogTypes.Hash) bool {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	if dag.pendedBatches.existTx(baseSeqHash, hash) {
		return true
	}
	return dag.accessor.ReadTransaction(hash) != nil
}

// GetTxByNonce gets tx from dag by sender's address and tx nonce
func (dag *Dag) GetTxByNonce(baseSeqHash ogTypes.Hash, addr ogTypes.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxByNonce(baseSeqHash, addr, nonce)
}

func (dag *Dag) getTxByNonce(baseSeqHash ogTypes.Hash, addr ogTypes.Address, nonce uint64) types.Txi {
	tx := dag.pendedBatches.getTxByNonce(baseSeqHash, addr, nonce)
	if tx != nil {
		return tx
	}
	return dag.accessor.ReadTxByNonce(addr, nonce)
}

// GetTxs get a bundle of txs according to a hash list.
func (dag *Dag) GetTxis(hashs []ogTypes.Hash) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxis(hashs)
}

func (dag *Dag) getTxis(hashs []ogTypes.Hash) types.Txis {
	var txs types.Txis
	for _, hash := range hashs {
		tx := dag.getTx(hash)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs
}

//func (dag *Dag) getTxisByType(hashs []ogTypes.Hash, baseType types.TxBaseType) types.Txis {
//	var txs types.Txis
//	for _, hash := range hashs {
//		tx := dag.getTxAndReceipt(hash)
//		if tx != nil && tx.GetType() == baseType {
//			txs = append(txs, tx)
//		}
//	}
//	return txs
//}

// GetTxConfirmHeight returns the height of sequencer that confirm this tx.
func (dag *Dag) GetTxConfirmHeight(hash ogTypes.Hash) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxConfirmHeight(hash)
}

func (dag *Dag) getTxConfirmHeight(hash ogTypes.Hash) (uint64, error) {
	tx := dag.getTx(hash)
	if tx == nil {
		return 0, fmt.Errorf("hash not exists: %s", hash)
	}
	return tx.GetBase().GetHeight(), nil
}

func (dag *Dag) GetTxisByHeight(height uint64) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	if height > dag.latestSequencer.GetHeight() {
		return nil
	}
	baseSeqHash := dag.latestSequencer.GetTxHash()
	txs := dag.pendedBatches.getTxsByHeight(baseSeqHash, height)
	if txs != nil {
		return txs
	}

	hashs, err := dag.accessor.ReadIndexedTxHashs(height)
	if err != nil {
		log.WithError(err).WithField("height", height).Trace("hashes not found")
		return nil
	}
	if len(hashs) == 0 {
		return nil
	}
	log.WithField("len tx ", len(hashs)).WithField("height", height).Trace("get txs")
	return dag.getTxis(hashs)
}

//func (dag *Dag) GetTxsByNumberAndType(height uint64, txType types.TxBaseType) types.Txis {
//	dag.mu.RLock()
//	defer dag.mu.RUnlock()
//
//	hashs := dag.getTxsHashesByHeight(height)
//	if hashs == nil {
//		return nil
//	}
//	if len(hashs) == 0 {
//		return nil
//	}
//	log.WithField("len tx ", len(hashs)).WithField("height", height).Trace("get txs")
//	return dag.getTxisByType(hashs, txType)
//}

func (dag *Dag) GetReceipt(hash ogTypes.Hash) *Receipt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	_, receipt := dag.pendedBatches.getTxAndReceipt(hash)
	if receipt != nil {
		return receipt
	}
	tx := dag.getConfirmedTx(hash)
	if tx == nil {
		return nil
	}
	seqid := tx.GetHeight()
	return dag.accessor.ReadReceipt(seqid, hash)
}

func (dag *Dag) GetSequencerByHash(hash ogTypes.Hash) *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *types.Sequencer:
		return tx
	default:
		return nil
	}
}

// GetSequencerByHeight only support those confirmed sequencers
func (dag *Dag) GetSequencerByHeight(height uint64) *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerByHeight(height)
}

func (dag *Dag) getSequencerByHeight(height uint64) *types.Sequencer {
	if height == 0 {
		return dag.genesis
	}
	if height > dag.latestSequencer.Height {
		return nil
	}
	baseSeqHash := dag.latestSequencer.GetTxHash()
	seq := dag.pendedBatches.getSeqByHeight(baseSeqHash, height)
	if seq != nil {
		return seq
	}
	seq, err := dag.accessor.ReadSequencerByHeight(height)
	if err != nil || seq == nil {
		log.WithField("height", height).WithError(err).Warn("head not found")
		return nil
	}
	return seq
}

//func (dag *Dag) GetSequencerHashByHeight(height uint64) ogTypes.Hash {
//	dag.mu.RLock()
//	defer dag.mu.RUnlock()
//
//	return dag.getSequencerHashByHeight(height)
//}
//
//func (dag *Dag) getSequencerHashByHeight(height uint64) ogTypes.Hash {
//	seq, err := dag.accessor.ReadSequencerByHeight(height)
//	if err != nil || seq == nil {
//		log.WithField("height", height).Warn("head not found")
//		return nil
//	}
//	hash := seq.GetTxHash()
//	return hash
//}

func (dag *Dag) GetSequencer(hash ogTypes.Hash, seqHeight uint64) *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *types.Sequencer:
		if tx.Height != seqHeight {
			log.Warn("seq height mismatch ")
			return nil
		}
		return tx
	default:
		return nil
	}
}

//func (dag *Dag) GetConfirmTime(seqHeight uint64) *types.ConfirmTime {
//
//	dag.mu.RLock()
//	defer dag.mu.RUnlock()
//	return dag.getConfirmTime(seqHeight)
//}
//
//func (dag *Dag) getConfirmTime(seqHeight uint64) *types.ConfirmTime {
//	if seqHeight == 0 {
//		return nil
//	}
//	cf := dag.accessor.readConfirmTime(seqHeight)
//	if cf == nil {
//		log.Warn("ConfirmTime not found")
//	}
//	return cf
//}

//func (dag *Dag) GetTxsHashesByNumber(Height uint64) []ogTypes.Hash {
//	dag.mu.RLock()
//	defer dag.mu.RUnlock()
//
//	return dag.getTxsHashesByHeight(Height)
//}
//
//func (dag *Dag) getTxsHashesByHeight(height uint64) []ogTypes.Hash {
//	if height > dag.latestSequencer.Number() {
//		return nil
//	}
//	hashs, err := dag.accessor.ReadIndexedTxHashs(height)
//	if err != nil {
//		log.WithError(err).WithField("height", height).Trace("hashes not found")
//	}
//	return hashs
//}

func (dag *Dag) GetBalance(baseSeqHash ogTypes.Hash, addr ogTypes.Address, tokenID int32) *math.BigInt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getBalance(baseSeqHash, addr, tokenID)
}

func (dag *Dag) getBalance(baseSeqHash ogTypes.Hash, addr ogTypes.Address, tokenID int32) *math.BigInt {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return nil
	}
	return db.GetTokenBalance(addr, tokenID)
}

func (dag *Dag) GetAllTokenBalance(baseSeqHash ogTypes.Hash, addr ogTypes.Address) state.BalanceSet {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getAlltokenBalance(baseSeqHash, addr)
}

func (dag *Dag) getAlltokenBalance(baseSeqHash ogTypes.Hash, addr ogTypes.Address) state.BalanceSet {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return nil
	}
	return db.GetAllTokenBalance(addr)
}

func (dag *Dag) GetToken(baseSeqHash ogTypes.Hash, tokenId int32) *state.TokenObject {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getToken(baseSeqHash, tokenId)
}

func (dag *Dag) getToken(baseSeqHash ogTypes.Hash, tokenId int32) *state.TokenObject {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return nil
	}
	if tokenId > db.LatestTokenID() {
		return nil
	}
	return db.GetTokenObject(tokenId)
}

func (dag *Dag) GetLatestTokenId(baseSeqHash ogTypes.Hash) int32 {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getLatestTokenId(baseSeqHash)
}

func (dag *Dag) getLatestTokenId(baseSeqHash ogTypes.Hash) int32 {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return -1
	}
	return db.LatestTokenID()
}

func (dag *Dag) GetTokens(baseSeqHash ogTypes.Hash) []*state.TokenObject {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTokens(baseSeqHash)
}

func (dag *Dag) getTokens(baseSeqHash ogTypes.Hash) []*state.TokenObject {
	tokens := make([]*state.TokenObject, 0)
	lid := dag.getLatestTokenId(baseSeqHash)

	for i := int32(0); i <= lid; i++ {
		token := dag.getToken(baseSeqHash, i)
		tokens = append(tokens, token)
	}
	return tokens
}

// GetLatestNonce returns the latest tx of an address.
func (dag *Dag) GetLatestNonce(baseSeqHash ogTypes.Hash, addr ogTypes.Address) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getLatestNonce(baseSeqHash, addr)
}

func (dag *Dag) getLatestNonce(baseSeqHash ogTypes.Hash, addr ogTypes.Address) (uint64, error) {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return 0, fmt.Errorf("base sequencer not found: %s", baseSeqHash.Hex())
	}
	return db.GetNonce(addr), nil
}

// GetState get contract's state from statedb.
func (dag *Dag) GetState(baseSeqHash ogTypes.Hash, addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getState(baseSeqHash, addr, key)
}

func (dag *Dag) getState(baseSeqHash ogTypes.Hash, addr ogTypes.Address, key ogTypes.Hash) ogTypes.Hash {
	db := dag.getDB(baseSeqHash)
	if db == nil {
		return nil
	}
	return db.GetState(addr, key)
}

func (dag *Dag) getDB(baseSeqHash ogTypes.Hash) *state.StateDB {
	if baseSeqHash.Cmp(dag.ConfirmedSequencerHash()) == 0 {
		return dag.confirmedStateDB
	}
	confirmBatch := dag.pendedBatches.getConfirmBatch(baseSeqHash)
	if confirmBatch != nil {
		return confirmBatch.db
	}
	return nil
}

////GetTxsByAddress get all txs from this address
//func (dag *Dag) GetTxsByAddress(addr ogTypes.Address) []types.Txi {
//	dag.mu.RLock()
//	defer dag.mu.RUnlock()
//
//	return dag.getTxsByAddress(addr)
//}
//
//func (dag *Dag) getTxsByAddress(addr ogTypes.Address) []types.Txi {
//	nonce, err := dag.getLatestNonce(addr)
//	if (err != nil) || (nonce == 0) {
//		return nil
//	}
//	var i int64
//	var txs []types.Txi
//	for i = int64(nonce); i > 0; i-- {
//		tx := dag.getTxByNonce(addr, uint64(i))
//		if tx != nil {
//			txs = append(txs, tx)
//		}
//	}
//	if len(txs) == 0 {
//		return nil
//	}
//	return txs
//}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) speculate(pushBatch *PushBatch) (ogTypes.Hash, error) {
	log.Tracef("speculate the batch: %s", pushBatch.String())

	confirmBatch, err := dag.process(pushBatch)
	if err != nil {
		return nil, err
	}
	dag.cachedBatches.purePush(confirmBatch.db.Root(), confirmBatch)
	return confirmBatch.db.Root(), nil
}

func (dag *Dag) push(pushBatch *PushBatch) error {
	log.Tracef("push the pushBatch: %s", pushBatch.String())

	var err error
	seq := pushBatch.Seq

	//var confirmBatch *ConfirmBatch
	confirmBatch := dag.cachedBatches.getConfirmBatch(seq.StateRoot)
	if confirmBatch == nil {
		confirmBatch, err = dag.process(pushBatch)
		if err != nil {
			return err
		}
	} else {
		confirmBatch.seq = seq
		dag.cachedBatches.pureDelete(seq.StateRoot)
	}

	root := confirmBatch.db.Root()

	// flush triedb into diskdb.
	triedb := confirmBatch.db.Database().TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		return fmt.Errorf("can't flush trie from triedb to diskdb, err: %v", err)
	}

	// change chosen batch if current is higher than chosen one
	if seq.Height > dag.latestSequencer.Height {
		dag.latestSequencer = seq
		dag.latestStateDB = confirmBatch.db
	}
	dag.pendedBatches.push(seq.GetTxHash(), confirmBatch)

	log.Tracef("successfully store seq: %s", seq.GetTxHash())
	log.WithField("height", seq.Height).WithField("txs number ", len(pushBatch.Txs)).Info("new height")
	return nil
}

func (dag *Dag) commit(seq *types.Sequencer) (err error) {
	confirmBatch := dag.pendedBatches.getConfirmBatch(seq.GetTxHash())
	if confirmBatch == nil {
		return fmt.Errorf("can't find pended seq: %s", seq.GetTxHash().Hex())
	}

	err = dag.flushAllToDB(confirmBatch)
	if err != nil {
		return err
	}

	dag.pendedBatches.confirm(confirmBatch)
	dag.confirmedSequencer = seq
	return nil
}

func (dag *Dag) process(pushBatch *PushBatch) (*ConfirmBatch, error) {
	var err error
	seq := pushBatch.Seq
	sort.Sort(pushBatch.Txs)

	parentSeqI := dag.getTx(seq.GetParentSeqHash())
	if parentSeqI != nil && parentSeqI.GetType() != types.TxBaseTypeSequencer {
		return nil, fmt.Errorf("parent sequencer not exists: %s", seq.GetParentSeqHash().Hex())
	}
	parentSeq := parentSeqI.(*types.Sequencer)
	confirmBatch, err := newConfirmBatch(seq, dag.db, parentSeq.StateRoot)
	if err != nil {
		return nil, err
	}

	// store the tx and update the state
	receipts := make(ReceiptSet)
	for _, txi := range pushBatch.Txs {
		txi.GetBase().Height = seq.Height
		receipt, err := dag.processTransaction(confirmBatch.db, seq, txi)
		if err != nil {
			return nil, err
		}
		confirmBatch.addTx(txi)
		receipts[txi.GetTxHash().HashKey()] = receipt
		log.WithField("tx", txi).Tracef("successfully process tx")
	}
	// process sequencer
	seqReceipt, err := dag.processTransaction(confirmBatch.db, seq, seq)
	if err != nil {
		return nil, err
	}

	confirmBatch.txReceipts = receipts
	confirmBatch.seqReceipt = seqReceipt

	// commit statedb's changes to trie and triedb
	_, err = confirmBatch.db.Commit()
	if err != nil {
		return nil, fmt.Errorf("can't Commit statedb, err: %v", err)
	}
	confirmBatch.db.ClearJournalAndRefund()

	return confirmBatch, nil
}

func (dag *Dag) flushAllToDB(confirmBatch *ConfirmBatch) (err error) {
	dbBatch := dag.accessor.NewBatch()

	// write txs
	var txHashes []ogTypes.Hash
	for _, tx := range confirmBatch.elders {
		err = dag.writeTransaction(dbBatch, tx)
		if err != nil {
			return err
		}
		txHashes = append(txHashes, tx.GetTxHash())
	}
	// store the hashs of the txs confirmed by this sequencer.
	if len(txHashes) > 0 {
		dag.accessor.WriteIndexedTxHashs(dbBatch, confirmBatch.seq.Height, txHashes)
	}
	// write sequencer
	err = dag.writeSequencer(dbBatch, confirmBatch.seq)
	if err != nil {
		return err
	}
	// write receipts
	confirmBatch.txReceipts[confirmBatch.seq.GetTxHash().HashKey()] = confirmBatch.seqReceipt
	err = dag.accessor.WriteReceipts(dbBatch, confirmBatch.seq.Height, confirmBatch.txReceipts)
	if err != nil {
		return err
	}

	err = dbBatch.Write()
	if err != nil {
		log.WithError(err).Warn("dbBatch write error")
		return err
	}
	return nil
}

// writeTransaction write the tx or sequencer into ogdb. It first writes
// the latest nonce of the tx's sender, then write the ([address, nonce] -> hash)
// relation into db, finally write the tx itself. Data will be overwritten
// if it already exists in db.
func (dag *Dag) writeTransaction(putter *Putter, tx types.Txi) error {
	// Write tx hash. This is aimed to allow users to query tx hash
	// by sender address and tx nonce.
	if tx.GetType() != types.TxBaseTypeArchive {
		err := dag.accessor.WriteTxHashByNonce(putter, tx.Sender(), tx.GetNonce(), tx.GetTxHash())
		if err != nil {
			return fmt.Errorf("write latest nonce err: %v", err)
		}
	}

	// Write tx itself
	err := dag.accessor.WriteTransaction(putter, tx)
	if err != nil {
		return err
	}

	//dag.txcached.add(tx)
	return nil
}

func (dag *Dag) deleteTransaction(hash ogTypes.Hash) error {
	return dag.accessor.DeleteTransaction(hash)
}

// processTransaction execute the tx and update the data in statedb.
func (dag *Dag) processTransaction(stateDB *state.StateDB, seq *types.Sequencer, tx types.Txi) (*Receipt, error) {
	txReceipt, err := dag.txProcessor.Process(stateDB, tx)
	if err != nil {
		return txReceipt, err
	}

	if !dag.vmProcessor.CanProcess(tx) {
		return txReceipt, nil
	}
	return dag.vmProcessor.Process(stateDB, tx, seq.Height)
}

// writeSequencer flushes sequencer into db indexed by seq hash and seq height
func (dag *Dag) writeSequencer(putter *Putter, seq *types.Sequencer) (err error) {
	err = dag.writeTransaction(putter, seq)
	if err != nil {
		return err
	}
	err = dag.accessor.WriteSequencerByHeight(putter, seq)
	if err != nil {
		return err
	}
	// set latest sequencer
	err = dag.accessor.WriteLatestConfirmedSeq(nil, seq)
	if err != nil {
		return err
	}
	return nil
}

// CallContract calls contract but disallow any modifications on
// statedb. This method will call ovm.StaticCall() to satisfy this.
func (dag *Dag) CallContract(addr ogTypes.Address20, data []byte) ([]byte, error) {
	// create ovm object.
	//
	// TODO gaslimit not implemented yet.
	vmContext := ovm.NewOVMContext(&ovm.DefaultChainContext{}, DefaultCoinbase, dag.latestStateDB)
	txContext := &ovm.TxContext{
		From:       DefaultCoinbase,
		Value:      math.NewBigInt(0),
		Data:       data,
		GasPrice:   math.NewBigInt(0),
		GasLimit:   DefaultGasLimit,
		Coinbase:   DefaultCoinbase,
		SequenceID: dag.latestSequencer.Height,
	}
	// TODO more interpreters should be initialized, here only evm.
	evmInterpreter := evm.NewEVMInterpreter(vmContext, txContext,
		&evm.InterpreterConfig{
			Debug: false,
		})
	ovmconf := &ovm.OVMConfig{
		NoRecursion: false,
	}
	ogvm := ovm.NewOVM(vmContext, []ovm.Interpreter{evmInterpreter}, ovmconf)

	ret, _, err := ogvm.StaticCall(vmtypes.AccountRef(*txContext.From), addr, txContext.Data, txContext.GasLimit)
	return ret, err
}

func (dag *Dag) revert(snapShotID int) {
	dag.latestStateDB.RevertToSnapshot(snapShotID)
}

type txcached struct {
	maxsize int
	order   []ogTypes.Hash
	txs     map[ogTypes.HashKey]types.Txi
}

func newTxcached(maxsize int) *txcached {
	return &txcached{
		maxsize: maxsize,
		order:   make([]ogTypes.Hash, 0),
		txs:     make(map[ogTypes.HashKey]types.Txi),
	}
}

func (tc *txcached) get(hash ogTypes.Hash) types.Txi {
	return tc.txs[hash.HashKey()]
}

func (tc *txcached) add(tx types.Txi) {
	if tx == nil {
		return
	}
	if _, ok := tc.txs[tx.GetTxHash().HashKey()]; ok {
		return
	}
	if len(tc.order) >= tc.maxsize {
		fstHash := tc.order[0]
		delete(tc.txs, fstHash.HashKey())
		tc.order = tc.order[1:]
	}
	tc.order = append(tc.order, tx.GetTxHash())
	tc.txs[tx.GetTxHash().HashKey()] = tx
}

type PushBatch struct {
	Seq *types.Sequencer
	Txs types.Txis
}

func (c *PushBatch) String() string {
	return fmt.Sprintf("seq: %s, txs: [%s]", c.Seq.String(), c.Txs.String())
}
