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
	"bytes"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/status"
	"github.com/annchain/OG/types/tx_types"
	"sort"
	"strconv"
	"time"

	// "fmt"
	"sync"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	evm "github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	vmtypes "github.com/annchain/OG/vm/types"

	log "github.com/sirupsen/logrus"
)

var (
	// empty address is the address used for contract creation, it
	// is filled in [tx.To].
	emptyAddress = common.BytesToAddress(nil)

	DefaultGasLimit = uint64(10000000000)

	DefaultCoinbase = common.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")
)

type DagConfig struct {
	GenesisPath string
}

type Dag struct {
	conf DagConfig

	db ogdb.Database
	// testDb is for test only, should be deleted later.
	testDb    ogdb.Database
	accessor  *Accessor
	preloadDB *state.PreloadDB
	statedb   *state.StateDB

	genesis         *tx_types.Sequencer
	latestSequencer *tx_types.Sequencer

	txcached *txcached

	OnConsensusTXConfirmed chan []types.Txi
	close                  chan struct{}

	mu sync.RWMutex
}

func NewDag(conf DagConfig, stateDBConfig state.StateDBConfig, db ogdb.Database, testDb ogdb.Database) (*Dag, error) {
	dag := &Dag{}

	dag.conf = conf
	dag.db = db
	dag.testDb = testDb
	dag.accessor = NewAccessor(db)
	// TODO
	// default maxsize of txcached is 10000,
	// move this size to config later.
	dag.txcached = newTxcached(10000)
	dag.close = make(chan struct{})

	restart, root := dag.LoadLastState()

	log.Infof("the root loaded from last state is: %x", root.ToBytes())
	statedb, err := state.NewStateDB(stateDBConfig, state.NewDatabase(db), root)
	if err != nil {
		return nil, fmt.Errorf("create statedb err: %v", err)
	}
	dag.statedb = statedb

	preloadDB := state.NewPreloadDB(statedb.Database(), statedb)
	dag.preloadDB = preloadDB

	if restart && dag.GetHeight() > 0 && root.Empty() {
		panic("should not be empty hash. Database may be corrupted. Please clean datadir")
	}

	if !restart {
		// TODO use config to load the genesis
		seq, balance := DefaultGenesis(conf.GenesisPath)
		if err := dag.Init(seq, balance); err != nil {
			return nil, err
		}
	}
	return dag, nil
}

func DefaultDagConfig() DagConfig {
	return DagConfig{}
}

func (dag *Dag) Start() {
	log.Infof("Dag Start")

	//goroutine.New(dag.loop)
}

func (dag *Dag) Stop() {
	close(dag.close)

	dag.statedb.Stop()
	log.Infof("Dag Stopped")
}

// TODO
// This is a temp function to solve the not working problem when
// restart the node. The perfect solution is to load the root from
// latest sequencer every time restart the node.
func (dag *Dag) LoadStateRoot() common.Hash {
	return dag.accessor.ReadLastStateRoot()
}

// StateDatabase is for testing only
func (dag *Dag) StateDatabase() *state.StateDB {
	return dag.statedb
}

// Init inits genesis sequencer and genesis state of the network.
func (dag *Dag) Init(genesis *tx_types.Sequencer, genesisBalance map[common.Address]*math.BigInt) error {
	if genesis.Height != 0 {
		return fmt.Errorf("invalheight genesis: height is not zero")
	}
	var err error
	//dbBatch := dag.db.NewBatch()

	// init genesis
	err = dag.accessor.WriteGenesis(genesis)
	if err != nil {
		return err
	}
	// init latest sequencer
	err = dag.accessor.WriteLatestSequencer(nil, genesis)
	if err != nil {
		return err
	}

	err = dag.accessor.WriteSequencerByHeight(nil, genesis)
	if err != nil {
		return err
	}
	// store genesis as first tx
	err = dag.WriteTransaction(nil, genesis)
	if err != nil {
		return err
	}
	log.Tracef("successfully store genesis: %s", genesis)

	// init genesis balance
	for addr, value := range genesisBalance {
		//tx := &types.Tx{}
		//tx.To = addr
		//tx.Value = value
		//tx.Type = types.TxBaseTypeNormal
		//tx.GetBase().Hash = tx.CalcTxHash()
		//dag.WriteTransaction(dbBatch, tx)

		dag.statedb.SetBalance(addr, value)
	}

	dag.genesis = genesis
	dag.latestSequencer = genesis

	log.Infof("Dag finish init")
	return nil
}

// LoadLastState load genesis and latestsequencer data from ogdb.
// return false if there is no genesis stored in the db.
func (dag *Dag) LoadLastState() (bool, common.Hash) {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	genesis := dag.accessor.ReadGenesis()
	if genesis == nil {
		return false, common.Hash{}
	}
	dag.genesis = genesis
	seq := dag.accessor.ReadLatestSequencer()
	if seq == nil {
		dag.latestSequencer = genesis
	} else {
		dag.latestSequencer = seq
	}
	root := dag.latestSequencer.StateRoot

	return true, root
}

// Genesis returns the genesis tx of dag
func (dag *Dag) Genesis() *tx_types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.genesis
}

// LatestSequencer returns the latest sequencer stored in dag
func (dag *Dag) LatestSequencer() *tx_types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.latestSequencer
}

//GetHeight get cuurent height
func (dag *Dag) GetHeight() uint64 {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.latestSequencer.Height
}

// Accessor returns the db accessor of dag
func (dag *Dag) Accessor() *Accessor {
	return dag.accessor
}

// Push trys to move a tx from tx pool to dag db.
func (dag *Dag) Push(batch *ConfirmBatch) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	return dag.push(batch)
}

// PrePush simulates the action of pushing sequencer into Dag ledger. Simulates will
// store the changes into cache statedb. Once the same sequencer comes, the cached
// states will becomes regular ones.
func (dag *Dag) PrePush(batch *ConfirmBatch) (common.Hash, error) {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	return dag.prePush(batch)
}

// GetTx gets tx from dag network indexed by tx hash. This function querys
// ogdb only.
func (dag *Dag) GetTx(hash common.Hash) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTx(hash)
}
func (dag *Dag) getTx(hash common.Hash) types.Txi {
	tx := dag.txcached.get(hash)
	if tx != nil {
		return tx
	}
	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) Has(hash common.Hash) bool {
	return dag.GetTx(hash) != nil
}

func (dag *Dag) Exist(addr common.Address) bool {
	return dag.statedb.Exist(addr)
}

// GetTxByNonce gets tx from dag by sender's address and tx nonce
func (dag *Dag) GetTxByNonce(addr common.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxByNonce(addr, nonce)
}

func (dag *Dag) getTestTx(hash common.Hash) types.Txi {
	data, _ := dag.testDb.Get(transactionKey(hash))
	if len(data) == 0 {
		log.Info("tx not found")
		return nil
	}
	prefixLen := len(contentPrefixTransaction)
	prefix := data[:prefixLen]
	data = data[prefixLen:]
	if bytes.Equal(prefix, contentPrefixTransaction) {
		var tx tx_types.Tx
		_, err := tx.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal tx  error")
			return nil
		}
		return &tx
	}
	if bytes.Equal(prefix, contentPrefixSequencer) {
		var sq tx_types.Sequencer
		_, err := sq.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal tx  error")
			return nil
		}
		return &sq
	}
	if bytes.Equal(prefix, contentPrefixCampaign) {
		var cp tx_types.Campaign
		_, err := cp.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal camp error")
			return nil
		}
		return &cp
	}
	if bytes.Equal(prefix, contentPrefixTermChg) {
		var tc tx_types.TermChange
		_, err := tc.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal termchg error")
			return nil
		}
		return &tc
	}
	if bytes.Equal(prefix, contentPrefixArchive) {
		var ac tx_types.Archive
		_, err := ac.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal archive error")
			return nil
		}
		return &ac
	}
	log.Warn("unknown prefix")
	return nil
}

func (dag *Dag) GetTestTxByAddressAndNonce(addr common.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	if dag.testDb == nil {
		log.Info("testdb is nil")
		return nil
	}
	data, _ := dag.testDb.Get(txHashFlowKey(addr, nonce))
	if len(data) == 0 {
		log.Info("hash not found")
		return nil
	}
	hash := common.BytesToHash(data)
	return dag.getTestTx(hash)
}

func (dag *Dag) getTxByNonce(addr common.Address, nonce uint64) types.Txi {
	return dag.accessor.ReadTxByNonce(addr, nonce)
}

// GetTxs get a bundle of txs according to a hash list.
func (dag *Dag) GetTxis(hashs common.Hashes) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxis(hashs)
}

func (dag *Dag) getTxis(hashs common.Hashes) types.Txis {
	var txs types.Txis
	for _, hash := range hashs {
		tx := dag.getTx(hash)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (dag *Dag) getTxisByType(hashs common.Hashes, baseType types.TxBaseType) types.Txis {
	var txs types.Txis
	for _, hash := range hashs {
		tx := dag.getTx(hash)
		if tx != nil && tx.GetType() == baseType {
			txs = append(txs, tx)
		}
	}
	return txs
}

// GetTxConfirmHeight returns the height of sequencer that confirm this tx.
func (dag *Dag) GetTxConfirmHeight(hash common.Hash) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxConfirmHeight(hash)
}

func (dag *Dag) getTxConfirmHeight(hash common.Hash) (uint64, error) {
	tx := dag.getTx(hash)
	if tx == nil {
		return 0, fmt.Errorf("hash not exists: %s", hash)
	}
	return tx.GetBase().GetHeight(), nil
}

func (dag *Dag) GetTxisByNumber(height uint64) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	hashs := dag.getTxsHashesByNumber(height)
	if hashs == nil {
		return nil
	}
	if len(*hashs) == 0 {
		return nil
	}
	log.WithField("len tx ", len(*hashs)).WithField("height", height).Trace("get txs")
	return dag.getTxis(*hashs)
}

func (dag *Dag) GetTestTxisByNumber(height uint64) (types.Txis, *tx_types.Sequencer) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	data, _ := dag.testDb.Get(seqHeightKey(height))
	if len(data) == 0 {
		log.Warnf("seq height %d not found", height)
		return nil, nil
	}
	//if len(data) == 0 {
	//	 log.Warnf("sequencer with SeqHeight %d not found", height)
	//}
	var seq tx_types.Sequencer
	_, err := seq.UnmarshalMsg(data)
	if err != nil {
		log.WithError(err).Warn("unmarsahl error")
		return nil, nil
	}
	data, _ = dag.testDb.Get(txIndexKey(height))
	if len(data) == 0 {
		log.Warnf("tx hashs with seq height %d not found", height)
		return nil, &seq
	}
	var hashs common.Hashes
	_, err = hashs.UnmarshalMsg(data)
	if err != nil {
		log.WithError(err).Warn("unmarshal err")
		return nil, &seq
	}
	if hashs == nil || len(hashs) == 0 {
		return nil, &seq
	}
	log.WithField("len tx ", len(hashs)).WithField("height", height).Trace("get txs")
	var txs types.Txis
	for _, hash := range hashs {
		tx := dag.getTestTx(hash)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs, &seq
}

func (dag *Dag) GetTxsByNumberAndType(height uint64, txType types.TxBaseType) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	hashs := dag.getTxsHashesByNumber(height)
	if hashs == nil {
		return nil
	}
	if len(*hashs) == 0 {
		return nil
	}
	log.WithField("len tx ", len(*hashs)).WithField("height", height).Trace("get txs")
	return dag.getTxisByType(*hashs, txType)
}

func (dag *Dag) GetReceipt(hash common.Hash) *Receipt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	if tx == nil {
		return nil
	}
	seqid := tx.GetHeight()
	return dag.accessor.ReadReceipt(seqid, hash)
}

func (dag *Dag) GetSequencerByHash(hash common.Hash) *tx_types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *tx_types.Sequencer:
		return tx
	default:
		return nil
	}
}

func (dag *Dag) GetSequencerByHeight(height uint64) *tx_types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerByHeight(height)
}

func (dag *Dag) getSequencerByHeight(height uint64) *tx_types.Sequencer {
	if height == 0 {
		return dag.genesis
	}
	if height > dag.latestSequencer.Height {
		return nil
	}
	seq, err := dag.accessor.ReadSequencerByHeight(height)
	if err != nil || seq == nil {
		log.WithField("height", height).WithError(err).Warn("head not found")
		return nil
	}
	return seq
}

func (dag *Dag) GetSequencerHashByHeight(height uint64) *common.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerHashByHeight(height)
}

func (dag *Dag) getSequencerHashByHeight(height uint64) *common.Hash {
	seq, err := dag.accessor.ReadSequencerByHeight(height)
	if err != nil || seq == nil {
		log.WithField("height", height).Warn("head not found")
		return nil
	}
	hash := seq.GetTxHash()
	return &hash
}

func (dag *Dag) GetSequencer(hash common.Hash, seqHeight uint64) *tx_types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *tx_types.Sequencer:
		if tx.Height != seqHeight {
			log.Warn("seq height mismatch ")
			return nil
		}
		return tx
	default:
		return nil
	}
}

func (dag *Dag) GetConfirmTime(seqHeight uint64) *types.ConfirmTime {

	dag.mu.RLock()
	defer dag.mu.RUnlock()
	return dag.getConfirmTime(seqHeight)
}

func (dag *Dag) getConfirmTime(seqHeight uint64) *types.ConfirmTime {
	if seqHeight == 0 {
		return nil
	}
	cf := dag.accessor.readConfirmTime(seqHeight)
	if cf == nil {
		log.Warn("ConfirmTime not found")
	}
	return cf
}

func (dag *Dag) GetTxsHashesByNumber(Height uint64) *common.Hashes {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxsHashesByNumber(Height)
}

func (dag *Dag) getTxsHashesByNumber(Height uint64) *common.Hashes {
	if Height > dag.latestSequencer.Number() {
		return nil
	}
	hashs, err := dag.accessor.ReadIndexedTxHashs(Height)
	if err != nil {
		log.WithError(err).WithField("height", Height).Trace("hashes not found")
	}
	return hashs
}

// GetBalance read the confirmed balance of an address from ogdb.
func (dag *Dag) GetBalance(addr common.Address, tokenID int32) *math.BigInt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getBalance(addr, tokenID)
}

func (dag *Dag) getBalance(addr common.Address, tokenID int32) *math.BigInt {
	return dag.statedb.GetTokenBalance(addr, tokenID)
}

func (dag *Dag) GetAllTokenBalance(addr common.Address) state.BalanceSet {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getAlltokenBalance(addr)
}

func (dag *Dag) getAlltokenBalance(addr common.Address) state.BalanceSet {
	return dag.statedb.GetAllTokenBalance(addr)
}

func (dag *Dag) GetToken(tokenId int32) *state.TokenObject {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	if tokenId > dag.statedb.LatestTokenID() {
		return nil
	}
	return dag.getToken(tokenId)
}

func (dag *Dag) getToken(tokenId int32) *state.TokenObject {
	return dag.statedb.GetTokenObject(tokenId)
}

func (dag *Dag) GetLatestTokenId() int32 {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getLatestTokenId()
}

func (dag *Dag) getLatestTokenId() int32 {
	return dag.statedb.LatestTokenID()
}

func (dag *Dag) GetTokens() []*state.TokenObject {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTokens()
}

func (dag *Dag) getTokens() []*state.TokenObject {
	tokens := make([]*state.TokenObject, 0)
	lid := dag.getLatestTokenId()

	for i := int32(0); i <= lid; i++ {
		token := dag.getToken(i)
		tokens = append(tokens, token)
	}
	return tokens
}

// GetLatestNonce returns the latest tx of an addresss.
func (dag *Dag) GetLatestNonce(addr common.Address) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getLatestNonce(addr)
}

func (dag *Dag) getLatestNonce(addr common.Address) (uint64, error) {

	return dag.statedb.GetNonce(addr), nil
}

// GetState get contract's state from statedb.
func (dag *Dag) GetState(addr common.Address, key common.Hash) common.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getState(addr, key)
}

func (dag *Dag) getState(addr common.Address, key common.Hash) common.Hash {
	return dag.statedb.GetState(addr, key)
}

//GetTxsByAddress get all txs from this address
func (dag *Dag) GetTxsByAddress(addr common.Address) []types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxsByAddress(addr)
}

func (dag *Dag) getTxsByAddress(addr common.Address) []types.Txi {
	nonce, err := dag.getLatestNonce(addr)
	if (err != nil) || (nonce == 0) {
		return nil
	}
	var i int64
	var txs []types.Txi
	for i = int64(nonce); i > 0; i-- {
		tx := dag.getTxByNonce(addr, uint64(i))
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	if len(txs) == 0 {
		return nil
	}
	return txs
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) prePush(batch *ConfirmBatch) (common.Hash, error) {
	log.Tracef("prePush the batch: %s", batch.String())

	sort.Sort(batch.Txs)

	dag.preloadDB.Reset()
	for _, txi := range batch.Txs {
		_, _, err := dag.ProcessTransaction(txi, true)
		if err != nil {
			return common.Hash{}, fmt.Errorf("process tx error: %v", err)
		}
	}
	return dag.preloadDB.Commit()
}

func (dag *Dag) push(batch *ConfirmBatch) error {
	log.Tracef("push the batch: %s", batch.String())

	if dag.latestSequencer.Height+1 != batch.Seq.Height {
		return fmt.Errorf("last sequencer Height mismatch old %d, new %d", dag.latestSequencer.Height, batch.Seq.Height)
	}

	var err error

	// TODO batch is not used properly.
	dbBatch := dag.accessor.NewBatch()
	receipts := make(ReceiptSet)
	//var  dbBatch ogdb.Putter

	// store the tx and update the state
	sort.Sort(batch.Txs)
	txhashes := common.Hashes{}
	consTxs := []types.Txi{}
	sId := dag.statedb.Snapshot()

	for _, txi := range batch.Txs {
		txi.GetBase().Height = batch.Seq.Height

		_, receipt, err := dag.ProcessTransaction(txi, false)
		if err != nil {
			dag.Revert(sId, nil)
			log.WithField("sid ", sId).WithField("hash ", txi.GetTxHash()).WithError(err).Warn(
				"process tx error , revert to snap short")
			return err
		}
		receipts[txi.GetTxHash().Hex()] = receipt

		txhashes = append(txhashes, txi.GetTxHash())
		// TODO
		// Consensus related txs should not some specific types, should be
		// changed to a modular way.
		txType := txi.GetType()
		if txType == types.TxBaseTypeCampaign || txType == types.TxBaseTypeTermChange {
			consTxs = append(consTxs, txi)
		}
	}
	var writedTxs types.Txis
	for _, txi := range batch.Txs {
		err = dag.WriteTransaction(dbBatch, txi)
		if err != nil {
			log.WithError(err).Error("write tx error, normally never happen")
			dag.Revert(sId, writedTxs)
			return fmt.Errorf("write tx into db error: %v", err)
		}
		tx := txi
		writedTxs = append(writedTxs, tx)
		log.WithField("tx", txi).Tracef("successfully process tx")
	}

	// save latest sequencer into db
	batch.Seq.GetBase().Height = batch.Seq.Height
	err = dag.WriteTransaction(dbBatch, batch.Seq)
	if err != nil {
		return err
	}
	_, receipt, err := dag.ProcessTransaction(batch.Seq, false)
	if err != nil {
		return err
	}
	receipts[batch.Seq.GetTxHash().Hex()] = receipt

	// write receipts.
	err = dag.accessor.WriteReceipts(dbBatch, batch.Seq.Height, receipts)
	if err != nil {
		return err
	}

	// commit statedb's changes to trie and triedb
	root, errdb := dag.statedb.Commit()
	if errdb != nil {
		log.Errorf("can't Commit statedb, err: %v", errdb)
		return fmt.Errorf("can't Commit statedb, err: %v", errdb)
	}
	dag.statedb.ClearJournalAndRefund()

	// TODO
	// compare the state root between seq.StateRoot and root after committing statedb.
	batch.Seq.StateRoot = root
	//if root.Cmp(batch.Seq.StateRoot) != 0 {
	//	log.Errorf("the state root after processing all txs is not the same as the root in seq. "+
	//		"root in statedb: %x, root in seq: %x", root.Bytes, batch.Seq.StateRoot.Bytes)
	//	dag.statedb.RevertToSnapshot(sId)
	//	return fmt.Errorf("root not the same. root in statedb: %x, root in seq: %x", root.Bytes, batch.Seq.StateRoot.Bytes)
	//}

	// flush triedb into diskdb.
	triedb := dag.statedb.Database().TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		log.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
		dag.statedb.RevertToSnapshot(sId)
		return fmt.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
	}

	// store the hashs of the txs confirmed by this sequencer.
	if len(txhashes) > 0 {
		dag.accessor.WriteIndexedTxHashs(dbBatch, batch.Seq.Height, &txhashes)
	}

	err = dag.accessor.WriteSequencerByHeight(dbBatch, batch.Seq)
	if err != nil {
		return err
	}

	// set latest sequencer
	err = dag.accessor.WriteLatestSequencer(dbBatch, batch.Seq)
	if err != nil {
		return err
	}

	err = dbBatch.Write()
	if err != nil {
		log.WithError(err).Warn("dbbatch write error")
		return err
	}
	dag.latestSequencer = batch.Seq

	log.Tracef("successfully store seq: %s", batch.Seq.GetTxHash())

	// TODO: confirm time is for tps calculation, delete later.
	cf := types.ConfirmTime{
		SeqHeight:   batch.Seq.Height,
		TxNum:       uint64(len(txhashes)),
		ConfirmTime: time.Now().Format(time.RFC3339Nano),
	}
	dag.writeConfirmTime(&cf)

	// send consensus related txs.
	if len(consTxs) != 0 && dag.OnConsensusTXConfirmed != nil && !status.NodeStopped {
		log.WithField("txs ", consTxs).Trace("sending consensus txs")
		goroutine.New(func() {
			dag.OnConsensusTXConfirmed <- consTxs
			log.WithField("txs ", consTxs).Trace("sent consensus txs")
		})
	}
	log.Tracef("successfully update latest seq: %s", batch.Seq.GetTxHash())
	log.WithField("height", batch.Seq.Height).WithField("txs number ", len(txhashes)).Info("new height")

	return nil
}

func (dag *Dag) writeConfirmTime(cf *types.ConfirmTime) error {
	return dag.accessor.writeConfirmTime(cf)
}

func (dag *Dag) TestWriteConfirmTIme(cf *types.ConfirmTime) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.latestSequencer = tx_types.RandomSequencer()
	dag.latestSequencer.Height = cf.SeqHeight
	return dag.writeConfirmTime(cf)
}

func (dag *Dag) ReadConfirmTime(seqHeight uint64) *types.ConfirmTime {
	return dag.accessor.readConfirmTime(seqHeight)
}

// WriteTransaction write the tx or sequencer into ogdb. It first writes
// the latest nonce of the tx's sender, then write the ([address, nonce] -> hash)
// relation into db, finally write the tx itself. Data will be overwritten
// if it already exists in db.
func (dag *Dag) WriteTransaction(putter *Putter, tx types.Txi) error {
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

	dag.txcached.add(tx)
	return nil
}

func (dag *Dag) DeleteTransaction(hash common.Hash) error {
	return dag.accessor.DeleteTransaction(hash)
}

// ProcessTransaction execute the tx and update the data in statedb.
//
// Besides balance and nonce, if a tx is trying to create or call a
// contract, vm part will be initiated to handle this.
func (dag *Dag) ProcessTransaction(tx types.Txi, preload bool) ([]byte, *Receipt, error) {
	// update nonce
	if tx.GetType() == types.TxBaseTypeArchive {
		//receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusArchiveSuccess, "", emptyAddress)
		return nil, nil, nil
	}

	var db state.StateDBInterface
	if preload {
		db = dag.preloadDB
	} else {
		db = dag.statedb
	}

	curNonce := db.GetNonce(tx.Sender())
	if !db.Exist(tx.Sender()) || tx.GetNonce() > curNonce {
		db.SetNonce(tx.Sender(), tx.GetNonce())
	}

	if tx.GetType() == types.TxBaseTypeSequencer {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
		return nil, receipt, nil
	}
	if tx.GetType() == types.TxBaseTypeCampaign {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
		return nil, receipt, nil
	}
	if tx.GetType() == types.TxBaseTypeTermChange {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
		return nil, receipt, nil
	}
	if tx.GetType() == types.TxBaseAction {
		actionTx := tx.(*tx_types.ActionTx)
		receipt, err := dag.processTokenTransaction(actionTx)
		if err != nil {
			return nil, receipt, fmt.Errorf("process action tx error: %v", err)
		}
		return nil, receipt, nil
	}

	if tx.GetType() != types.TxBaseTypeNormal {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusUnknownTxType, "", emptyAddress)
		return nil, receipt, nil
	}

	// transfer balance
	txnormal := tx.(*tx_types.Tx)
	if txnormal.Value.Value.Sign() != 0 && !txnormal.To.EqualTo(emptyAddress) {
		db.SubTokenBalance(txnormal.Sender(), txnormal.TokenId, txnormal.Value)
		db.AddTokenBalance(txnormal.To, txnormal.TokenId, txnormal.Value)
	}
	// return when its not contract related tx.
	if len(txnormal.Data) == 0 {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
		return nil, receipt, nil
	}

	// create ovm object.
	//
	// TODO gaslimit not implemented yet.
	var vmContext *vmtypes.Context
	if preload {
		vmContext = ovm.NewOVMContext(&ovm.DefaultChainContext{}, &DefaultCoinbase, dag.preloadDB)
	} else {
		vmContext = ovm.NewOVMContext(&ovm.DefaultChainContext{}, &DefaultCoinbase, dag.statedb)
	}

	txContext := &ovm.TxContext{
		From:       txnormal.Sender(),
		Value:      txnormal.Value,
		Data:       txnormal.Data,
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

	var ret []byte
	//var leftOverGas uint64
	var contractAddress = emptyAddress
	var err error
	var receipt *Receipt
	if txnormal.To.Bytes == emptyAddress.Bytes {
		ret, contractAddress, _, err = ogvm.Create(vmtypes.AccountRef(txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	} else {
		ret, _, err = ogvm.Call(vmtypes.AccountRef(txContext.From), txnormal.To, txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	}
	if err != nil {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusOVMFailed, err.Error(), emptyAddress)
		log.WithError(err).Warn("vm processing error")
		return nil, receipt, fmt.Errorf("vm processing error: %v", err)
	}
	if txnormal.To.Bytes == emptyAddress.Bytes {
		receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, contractAddress.Hex(), contractAddress)
	} else {
		receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, fmt.Sprintf("%x", ret), contractAddress)
	}
	return ret, receipt, nil
}

func (dag *Dag) processTokenTransaction(tx *tx_types.ActionTx) (*Receipt, error) {

	actionData := tx.ActionData.(*tx_types.PublicOffering)
	if tx.Action == tx_types.ActionTxActionIPO {
		issuer := tx.Sender()
		name := actionData.TokenName
		reIssuable := actionData.EnableSPO
		amount := actionData.Value

		tokenID, err := dag.statedb.IssueToken(issuer, name, "", reIssuable, amount)
		if err != nil {
			receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		actionData.TokenId = tokenID
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil
	}
	if tx.Action == tx_types.ActionTxActionSPO {
		tokenID := actionData.TokenId
		amount := actionData.Value

		err := dag.statedb.ReIssueToken(tokenID, amount)
		if err != nil {
			receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil
	}
	if tx.Action == tx_types.ActionTxActionDestroy {
		tokenID := actionData.TokenId

		err := dag.statedb.DestroyToken(tokenID)
		if err != nil {
			receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil
	}

	err := fmt.Errorf("unknown tx action: %d", tx.Action)
	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
	return receipt, err
}

// CallContract calls contract but disallow any modifications on
// statedb. This method will call ovm.StaticCall() to satisfy this.
func (dag *Dag) CallContract(addr common.Address, data []byte) ([]byte, error) {
	// create ovm object.
	//
	// TODO gaslimit not implemented yet.
	vmContext := ovm.NewOVMContext(&ovm.DefaultChainContext{}, &DefaultCoinbase, dag.statedb)
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

	ret, _, err := ogvm.StaticCall(vmtypes.AccountRef(txContext.From), addr, txContext.Data, txContext.GasLimit)
	return ret, err
}

func (dag *Dag) Revert(snapShotID int, txs types.Txis) {
	dag.statedb.RevertToSnapshot(snapShotID)
	for _, txi := range txs {
		err := dag.DeleteTransaction(txi.GetTxHash())
		if err != nil {
			log.WithField("tx ", txi).WithError(err).Error("delete tx error")
		}
	}
}

type txcached struct {
	maxsize int
	order   common.Hashes
	txs     map[common.Hash]types.Txi
}

func newTxcached(maxsize int) *txcached {
	return &txcached{
		maxsize: maxsize,
		order:   common.Hashes{},
		txs:     make(map[common.Hash]types.Txi),
	}
}

func (tc *txcached) get(hash common.Hash) types.Txi {
	return tc.txs[hash]
}

func (tc *txcached) add(tx types.Txi) {
	if _, ok := tc.txs[tx.GetTxHash()]; ok {
		return
	}
	if len(tc.order) >= tc.maxsize {
		fstHash := tc.order[0]
		delete(tc.txs, fstHash)
		tc.order = tc.order[1:]
	}
	tc.order = append(tc.order, tx.GetTxHash())
	tc.txs[tx.GetTxHash()] = tx
}

type ConfirmBatch struct {
	Seq *tx_types.Sequencer
	Txs types.Txis
}

func (c *ConfirmBatch) String() string {
	return fmt.Sprintf("seq: %s, txs: [%s]", c.Seq.String(), c.Txs.String())
}
