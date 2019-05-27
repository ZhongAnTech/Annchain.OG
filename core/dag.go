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
	"sort"
	"time"

	// "fmt"
	"sync"

	"github.com/annchain/OG/common/crypto"
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
	emptyAddress = types.BytesToAddress(nil)

	DefaultGasLimit = uint64(10000000000)

	DefaultCoinbase = types.HexToAddress("0x1234567812345678AABBCCDDEEFF998877665544")
)

type DagConfig struct {
	GenesisPath string
}

type Dag struct {
	conf DagConfig

	db ogdb.Database
	// oldDb is for test only, should be deleted later.
	oldDb    ogdb.Database
	accessor *Accessor
	statedb  *state.StateDB

	genesis         *types.Sequencer
	latestSequencer *types.Sequencer

	txcached *txcached

	OnConsensusTXConfirmed chan []types.Txi

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, stateDBConfig state.StateDBConfig, db ogdb.Database, oldDb ogdb.Database, cryptoType crypto.CryptoType) (*Dag, error) {
	dag := &Dag{}

	dag.conf = conf
	dag.db = db
	dag.oldDb = oldDb
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

	if !restart {
		// TODO use config to load the genesis
		seq, balance := DefaultGenesis(cryptoType, conf.GenesisPath)
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
	dag.wg.Wait()
	// TODO
	// the state root should be stored in sequencer rather than stored seperately.
	dag.SaveStateRoot()
	dag.statedb.Stop()
	log.Infof("Dag Stopped")
}

// TODO
// This is a temp function to solve the not working problem when
// restart the node. The perfect solution is to load the root from
// latest sequencer every time restart the node.
func (dag *Dag) SaveStateRoot() {
	key := []byte("stateroot")
	dag.db.Put(key, dag.statedb.Root().ToBytes())
	log.Infof("stateroot saved: %x", dag.statedb.Root().ToBytes())
}

// TODO
// This is a temp function to solve the not working problem when
// restart the node. The perfect solution is to load the root from
// latest sequencer every time restart the node.
func (dag *Dag) LoadStateRoot() types.Hash {
	key := []byte("stateroot")
	rootbyte, _ := dag.db.Get(key)
	return types.BytesToHash(rootbyte)
}

// StateDatabase is for testing only
func (dag *Dag) StateDatabase() *state.StateDB {
	return dag.statedb
}

// Init inits genesis sequencer and genesis state of the network.
func (dag *Dag) Init(genesis *types.Sequencer, genesisBalance map[types.Address]*math.BigInt) error {
	if genesis.Height != 0 {
		return fmt.Errorf("invalheight genesis: height is not zero")
	}
	var err error
	dbBatch := dag.db.NewBatch()

	// init genesis
	err = dag.accessor.WriteGenesis(genesis)
	if err != nil {
		return err
	}
	// init latest sequencer
	err = dag.accessor.WriteLatestSequencer(genesis)
	if err != nil {
		return err
	}

	err = dag.accessor.WriteSequencerByHeight(genesis)
	if err != nil {
		return err
	}
	// store genesis as first tx
	err = dag.WriteTransaction(dbBatch, genesis)
	if err != nil {
		return err
	}
	log.Tracef("successfully store genesis: %s", genesis.String())

	// init genesis balance
	for addr, value := range genesisBalance {
		tx := &types.Tx{}
		tx.From = addr
		tx.Value = value
		tx.Type = types.TxBaseTypeNormal
		tx.GetBase().Hash = tx.CalcTxHash()
		dag.WriteTransaction(dbBatch, tx)

		dag.statedb.SetBalance(addr, value)
	}

	dag.genesis = genesis
	dag.latestSequencer = genesis

	log.Infof("Dag finish init")
	return nil
}

// LoadLastState load genesis and latestsequencer data from ogdb.
// return false if there is no genesis stored in the db.
func (dag *Dag) LoadLastState() (bool, types.Hash) {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	genesis := dag.accessor.ReadGenesis()
	if genesis == nil {
		return false, types.Hash{}
	}
	dag.genesis = genesis
	seq := dag.accessor.ReadLatestSequencer()
	if seq == nil {
		dag.latestSequencer = genesis
	} else {
		dag.latestSequencer = seq
	}
	root := dag.LoadStateRoot()

	return true, root
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
	dag.wg.Add(1)
	defer dag.wg.Done()

	return dag.push(batch)
}

// GetTx gets tx from dag network indexed by tx hash. This function querys
// ogdb only.
func (dag *Dag) GetTx(hash types.Hash) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTx(hash)
}
func (dag *Dag) getTx(hash types.Hash) types.Txi {
	tx := dag.txcached.get(hash)
	if tx != nil {
		return tx
	}
	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) Has(hash types.Hash) bool {
	return dag.GetTx(hash) != nil
}

func (dag *Dag) Exist(addr types.Address) bool {
	return dag.statedb.Exist(addr)
}

// GetTxByNonce gets tx from dag by sender's address and tx nonce
func (dag *Dag) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxByNonce(addr, nonce)
}

func (dag *Dag) GetOldTx(addr types.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	if dag.oldDb == nil {
		return nil
	}
	data, _ := dag.oldDb.Get(txHashFlowKey(addr, nonce))
	if len(data) == 0 {
		return nil
	}
	hash := types.BytesToHash(data)
	data, _ = dag.oldDb.Get(transactionKey(hash))
	if len(data) == 0 {
		return nil
	}
	prefixLen := len(contentPrefixTransaction)
	prefix := data[:prefixLen]
	data = data[prefixLen:]
	if bytes.Equal(prefix, contentPrefixTransaction) {
		var tx types.Tx
		_, err := tx.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal tx  error")
			return nil
		}
		return &tx
	}
	if bytes.Equal(prefix, contentPrefixSequencer) {
		var sq types.Sequencer
		_, err := sq.UnmarshalMsg(data)
		if err != nil {
			log.WithError(err).Warn("unmarshal tx  error")
			return nil
		}
		return &sq
	}
	return nil

}

func (dag *Dag) getTxByNonce(addr types.Address, nonce uint64) types.Txi {
	return dag.accessor.ReadTxByNonce(addr, nonce)
}

// GetTxs get a bundle of txs according to a hash list.
func (dag *Dag) GetTxis(hashs types.Hashes) types.Txis {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxis(hashs)
}

func (dag *Dag) getTxis(hashs types.Hashes) types.Txis {
	var txs types.Txis
	for _, hash := range hashs {
		tx := dag.getTx(hash)
		if tx != nil {
			txs = append(txs, tx)
		}
	}
	return txs
}

func (dag *Dag) getTxisByType(hashs types.Hashes, baseType types.TxBaseType) types.Txis {
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
func (dag *Dag) GetTxConfirmHeight(hash types.Hash) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxConfirmHeight(hash)
}
func (dag *Dag) getTxConfirmHeight(hash types.Hash) (uint64, error) {
	tx := dag.getTx(hash)
	if tx == nil {
		return 0, fmt.Errorf("hash not exists: %s", hash.String())
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

func (dag *Dag) GetReceipt(hash types.Hash) *Receipt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	if tx == nil {
		return nil
	}
	seqid := tx.GetHeight()
	return dag.accessor.ReadReceipt(seqid, hash)
}

func (dag *Dag) GetSequencerByHash(hash types.Hash) *types.Sequencer {
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
	seq, err := dag.accessor.ReadSequencerByHeight(height)
	if err != nil || seq == nil {
		log.WithField("height", height).WithError(err).Warn("head not found")
		return nil
	}
	return seq
}

func (dag *Dag) GetSequencerHashByHeight(height uint64) *types.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerHashByHeight(height)
}

func (dag *Dag) getSequencerHashByHeight(height uint64) *types.Hash {
	seq, err := dag.accessor.ReadSequencerByHeight(height)
	if err != nil || seq == nil {
		log.WithField("height", height).Warn("head not found")
		return nil
	}
	hash := seq.GetTxHash()
	return &hash
}

func (dag *Dag) GetSequencer(hash types.Hash, seqHeight uint64) *types.Sequencer {
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

func (dag *Dag) GetTxsHashesByNumber(Height uint64) *types.Hashes {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxsHashesByNumber(Height)
}

func (dag *Dag) getTxsHashesByNumber(Height uint64) *types.Hashes {
	if Height > dag.latestSequencer.Number() {
		return nil
	}
	hashs, err := dag.accessor.ReadIndexedTxHashs(Height)
	if err != nil {
		log.WithError(err).WithField("height", Height).Warn("head not found")
	}
	return hashs
}

// GetBalance read the confirmed balance of an address from ogdb.
func (dag *Dag) GetBalance(addr types.Address) *math.BigInt {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getBalance(addr)
}

func (dag *Dag) getBalance(addr types.Address) *math.BigInt {
	return dag.statedb.GetBalance(addr)
}

// GetLatestNonce returns the latest tx of an addresss.
func (dag *Dag) GetLatestNonce(addr types.Address) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getLatestNonce(addr)
}

func (dag *Dag) getLatestNonce(addr types.Address) (uint64, error) {
	if !dag.statedb.Exist(addr) {
		return uint64(0), types.ErrNonceNotExist
	}
	return dag.statedb.GetNonce(addr), nil
}

// GetState get contract's state from statedb.
func (dag *Dag) GetState(addr types.Address, key types.Hash) types.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getState(addr, key)
}

func (dag *Dag) getState(addr types.Address, key types.Hash) types.Hash {
	return dag.statedb.GetState(addr, key)
}

//GetTxsByAddress get all txs from this address
func (dag *Dag) GetTxsByAddress(addr types.Address) []types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxsByAddress(addr)
}

func (dag *Dag) getTxsByAddress(addr types.Address) []types.Txi {
	nonce, err := dag.getLatestNonce(addr)
	if (err != nil) && (err != types.ErrNonceNotExist) {
		return nil
	}
	var i int64
	var txs []types.Txi
	for i = int64(nonce); i >= 0; i-- {
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

func (dag *Dag) push(batch *ConfirmBatch) error {
	if dag.latestSequencer.Height+1 != batch.Seq.Height {
		return fmt.Errorf("last sequencer Height mismatch old %d, new %d", dag.latestSequencer.Height, batch.Seq.Height)
	}

	var err error

	// TODO batch is not used properly.
	dbBatch := dag.db.NewBatch()
	receipts := make(ReceiptSet)

	// store the tx and update the state
	sort.Sort(batch.Txs)
	txhashes := types.Hashes{}
	consTxs := []types.Txi{}
	for _, txi := range batch.Txs {
		txi.GetBase().Height = batch.Seq.Height
		err = dag.WriteTransaction(dbBatch, txi)
		if err != nil {
			return fmt.Errorf("write tx into db error: %v", err)
		}
		_, receipt, err := dag.ProcessTransaction(txi)
		if err != nil {
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
		log.WithField("tx", txi).Tracef("successfully process tx")
	}

	// save latest sequencer into db
	batch.Seq.GetBase().Height = batch.Seq.Height
	err = dag.WriteTransaction(dbBatch, batch.Seq)
	if err != nil {
		return err
	}
	_, receipt, err := dag.ProcessTransaction(batch.Seq)
	if err != nil {
		return err
	}
	receipts[batch.Seq.GetTxHash().Hex()] = receipt

	// write receipts.
	err = dag.accessor.WriteReceipts(batch.Seq.Height, receipts)
	if err != nil {
		return err
	}

	// TODO
	// get new trie root after commit, then compare new root
	// to the root in seq. If not equal then return error.

	// commit statedb's changes to trie and triedb
	root, errdb := dag.statedb.Commit()
	if errdb != nil {
		log.Errorf("can't Commit statedb, err: ", errdb)
		return fmt.Errorf("can't Commit statedb, err: %v", errdb)
	}
	// flush triedb into diskdb.
	triedb := dag.statedb.Database().TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		log.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
		return fmt.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
	}

	// store the hashs of the txs confirmed by this sequencer.
	if len(txhashes) > 0 {
		dag.accessor.WriteIndexedTxHashs(batch.Seq.Height, &txhashes)
	}
	err = dag.accessor.WriteSequencerByHeight(batch.Seq)
	if err != nil {
		return err
	}
	// set latest sequencer
	err = dag.accessor.WriteLatestSequencer(batch.Seq)
	if err != nil {
		return err
	}
	dag.latestSequencer = batch.Seq


	log.Tracef("successfully store seq: %s", batch.Seq.GetTxHash().String())

	// TODO: confirm time is for tps calculation, delete later.
	cf := types.ConfirmTime{
		SeqHeight:   batch.Seq.Height,
		TxNum:       uint64(len(txhashes)),
		ConfirmTime: time.Now().Format(time.RFC3339Nano),
	}
	dag.writeConfirmTime(&cf)

	// send consensus related txs.
	if len(consTxs) != 0 {
		log.WithField("txs ", consTxs).Trace("sending consensus txs")
		go func() {
			if dag.OnConsensusTXConfirmed != nil {
				dag.OnConsensusTXConfirmed <- consTxs
			}
			log.WithField("txs ", consTxs).Trace("sent consensus txs")
		}()
	}
	log.Tracef("successfully update latest seq: %s", batch.Seq.GetTxHash().String())
	log.WithField("height", batch.Seq.Height).WithField("txs number ", len(txhashes)).Info("new height")

	return nil
}

func (dag *Dag) writeConfirmTime(cf *types.ConfirmTime) error {
	return dag.accessor.writeConfirmTime(cf)
}

func (dag *Dag) ReadConfirmTime(seqHeight uint64) *types.ConfirmTime {
	return dag.accessor.readConfirmTime(seqHeight)
}

// WriteTransaction write the tx or sequencer into ogdb. It first writes
// the latest nonce of the tx's sender, then write the ([address, nonce] -> hash)
// relation into db, finally write the tx itself. Data will be overwritten
// if it already exists in db.
func (dag *Dag) WriteTransaction(putter ogdb.Putter, tx types.Txi) error {
	// Write tx hash. This is aimed to allow users to query tx hash
	// by sender address and tx nonce.
	err := dag.accessor.WriteTxHashByNonce(tx.Sender(), tx.GetNonce(), tx.GetTxHash())
	if err != nil {
		return fmt.Errorf("write latest nonce err: %v", err)
	}

	// Write tx itself
	err = dag.accessor.WriteTransaction(putter, tx)
	if err != nil {
		return err
	}

	dag.txcached.add(tx)
	return nil
}

// ProcessTransaction execute the tx and update the data in statedb.
//
// Besides balance and nonce, if a tx is trying to create or call a
// contract, vm part will be initiated to handle this.
func (dag *Dag) ProcessTransaction(tx types.Txi) ([]byte, *Receipt, error) {
	// update nonce
	curNonce := dag.statedb.GetNonce(tx.Sender())
	if !dag.statedb.Exist(tx.Sender()) || tx.GetNonce() > curNonce {
		dag.statedb.SetNonce(tx.Sender(), tx.GetNonce())
	}

	if tx.GetType() == types.TxBaseTypeSequencer {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSeqSuccess, "", emptyAddress)
		return nil, receipt, nil
	}
	if tx.GetType() == types.TxBaseTypeCampaign {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusCampaignSuccess, "", emptyAddress)
		return nil, receipt, nil
	}
	if tx.GetType() == types.TxBaseTypeTermChange {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusTermChangeSuccess, "", emptyAddress)
		return nil, receipt, nil
	}

	// transfer balance
	txnormal := tx.(*types.Tx)
	if txnormal.Value.Value.Sign() != 0 {
		dag.statedb.SubBalance(txnormal.From, txnormal.Value)
		dag.statedb.AddBalance(txnormal.To, txnormal.Value)
	}
	// return when its not contract related tx.
	if len(txnormal.Data) == 0 {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusTxSuccess, "", emptyAddress)
		return nil, receipt, nil
	}

	// create ovm object.
	//
	// TODO gaslimit not implemented yet.
	vmContext := ovm.NewOVMContext(&ovm.DefaultChainContext{}, &DefaultCoinbase, dag.statedb)
	txContext := &ovm.TxContext{
		From:       txnormal.From,
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
	var leftOverGas uint64
	var contractAddress = emptyAddress
	var err error
	if txnormal.To.Bytes == emptyAddress.Bytes {
		ret, contractAddress, leftOverGas, err = ogvm.Create(vmtypes.AccountRef(txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	} else {
		ret, leftOverGas, err = ogvm.Call(vmtypes.AccountRef(txContext.From), txnormal.To, txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	}
	if err != nil {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusOVMFailed, err.Error(), emptyAddress)
		return nil, receipt, fmt.Errorf("vm processing error: %v", err)
	}
	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusTxSuccess, "", contractAddress)

	// TODO
	// not finished yet
	//
	// 1. refund gas
	// 2. add service fee to coinbase
	log.Debugf("leftOverGas not used yet, this log is for compiling, ret: %x, leftOverGas: %d", ret, leftOverGas)
	return ret, receipt, nil
}

// CallContract calls contract but disallow any modifications on
// statedb. This method will call ovm.StaticCall() to satisfy this.
func (dag *Dag) CallContract(addr types.Address, data []byte) ([]byte, error) {
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

// Finalize
func (dag *Dag) Finalize() error {
	// consensus
	// TODO

	// state

	// TODO
	// get new trie root after commit, then compare new root
	// to the root in seq. If not equal then return error.

	// commit statedb's changes to trie and triedb
	root, errdb := dag.statedb.Commit()
	if errdb != nil {
		log.Errorf("can't Commit statedb, err: ", errdb)
		return fmt.Errorf("can't Commit statedb, err: %v", errdb)
	}
	// flush triedb into diskdb.
	triedb := dag.statedb.Database().TrieDB()
	err := triedb.Commit(root, false)
	if err != nil {
		log.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
		return fmt.Errorf("can't flush trie from triedb into diskdb, err: %v", err)
	}

	return nil
}

type txcached struct {
	maxsize int
	order   types.Hashes
	txs     map[types.Hash]types.Txi
}

func newTxcached(maxsize int) *txcached {
	return &txcached{
		maxsize: maxsize,
		order:   types.Hashes{},
		txs:     make(map[types.Hash]types.Txi),
	}
}

func (tc *txcached) get(hash types.Hash) types.Txi {
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
	Seq *types.Sequencer
	Txs types.Txis
}
