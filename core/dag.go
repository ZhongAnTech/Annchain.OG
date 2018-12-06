package core

import (
	"fmt"
	"sort"

	// "fmt"
	"sync"
	"time"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type DagConfig struct{}

type Dag struct {
	conf DagConfig

	db       ogdb.Database
	accessor *Accessor
	statedb  *StateDB

	genesis        *types.Sequencer
	latestSeqencer *types.Sequencer

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, db ogdb.Database) *Dag {
	dag := &Dag{}

	stateDBConfig := StateDBConfig{
		FlushTimer:     time.Duration(viper.GetInt("statedb.flush_timer_s")),
		PurgeTimer:     time.Duration(viper.GetInt("statedb.purge_timer_s")),
		BeatExpireTime: time.Second * time.Duration(viper.GetInt("statedb.beat_expire_time_s")),
	}

	dag.conf = conf
	dag.db = db
	dag.accessor = NewAccessor(db)
	dag.statedb = NewStateDB(stateDBConfig, db, dag.accessor)
	dag.close = make(chan struct{})

	return dag
}

func DefaultDagConfig() DagConfig {
	return DagConfig{}
}

type ConfirmBatch struct {
	Seq      *types.Sequencer
	Batch    map[types.Address]*BatchDetail
	TxHashes *types.Hashs
}

// BatchDetail describes all the details of a specific address within a
// sequencer confirmation term.
// - TxList - represents the txs sent by this addrs, ordered by nonce.
// - Neg - means the amount this address should spent out.
// - Pos - means the amount this address get paid.
type BatchDetail struct {
	TxList *TxList
	Neg    *math.BigInt
	Pos    *math.BigInt
}

func (dag *Dag) Start() {
	log.Infof("Dag Start")

	// go dag.loop()
}

func (dag *Dag) Stop() {
	close(dag.close)
	dag.wg.Wait()
	dag.statedb.Stop()
	log.Infof("Dag Stopped")
}

// Init inits genesis sequencer and genesis state of the network.
func (dag *Dag) Init(genesis *types.Sequencer, genesisBalance map[types.Address]*math.BigInt) error {
	if genesis.Id != 0 {
		return fmt.Errorf("invalid genesis: id is not zero")
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

	err = dag.accessor.WriteSequencerById(genesis)
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
		err = dag.accessor.SetBalance(addr, value)
		if err != nil {
			log.Errorf("init genesis balance error, addr: %s, balance: %s, error: %v",
				addr.String(), value.String(), err)
		}
	}

	dag.genesis = genesis
	dag.latestSeqencer = genesis

	log.Infof("Dag finish init")
	return nil
}

// LoadLastState load genesis and latestsequencer data from ogdb. return false if there
// is no genesis stored in the db.
func (dag *Dag) LoadLastState() bool {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	genesis := dag.accessor.ReadGenesis()
	if genesis == nil {
		return false
	}
	dag.genesis = genesis
	seq := dag.accessor.ReadLatestSequencer()
	if seq == nil {
		dag.latestSeqencer = genesis
	} else {
		dag.latestSeqencer = seq
	}

	return true
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

	return dag.latestSeqencer
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
	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) Has(hash types.Hash) bool {
	return dag.GetTx(hash) != nil
}

// GetTxByNonce gets tx from dag by sender's address and tx nonce
func (dag *Dag) GetTxByNonce(addr types.Address, nonce uint64) types.Txi {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxByNonce(addr, nonce)
}
func (dag *Dag) getTxByNonce(addr types.Address, nonce uint64) types.Txi {
	return dag.accessor.ReadTxByNonce(addr, nonce)
}

// GetTxs get a bundle of txs according to a hash list.
func (dag *Dag) GetTxs(hashs []types.Hash) []*types.Tx {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxs(hashs)
}
func (dag *Dag) getTxs(hashs []types.Hash) []*types.Tx {
	var txs []*types.Tx
	for _, hash := range hashs {
		tx := dag.getTx(hash)
		switch tx := tx.(type) {
		case *types.Tx:
			txs = append(txs, tx)
		}
	}
	return txs
}

// GetTxConfirmId returns the id of the sequencer that confirm this tx.
func (dag *Dag) GetTxConfirmId(hash types.Hash) (uint64, error) {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxConfirmId(hash)
}
func (dag *Dag) getTxConfirmId(hash types.Hash) (uint64, error) {
	return dag.accessor.ReadTxSeqRelation(hash)
}

func (dag *Dag) GetTxsByNumber(id uint64) []*types.Tx {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	hashs := dag.getTxsHashesByNumber(id)
	if hashs == nil {
		return nil
	}
	if len(*hashs) == 0 {
		return nil
	}
	log.WithField("len tx ", len(*hashs)).WithField("id", id).Trace("get txs")
	return dag.getTxs(*hashs)
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

func (dag *Dag) GetSequencerById(id uint64) *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerById(id)
}

func (dag *Dag) getSequencerById(id uint64) *types.Sequencer {
	if id == 0 {
		return dag.genesis
	}
	if id > dag.latestSeqencer.Id {
		return nil
	}
	seq, err := dag.accessor.ReadSequencerById(id)
	if err != nil || seq == nil {
		log.WithField("id", id).WithError(err).Warn("head not found")
		return nil
	}
	return seq
}

func (dag *Dag) GetSequencerHashById(id uint64) *types.Hash {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getSequencerHashById(id)
}

func (dag *Dag) getSequencerHashById(id uint64) *types.Hash {
	seq, err := dag.accessor.ReadSequencerById(id)
	if err != nil || seq == nil {
		log.WithField("id", id).Warn("head not found")
		return nil
	}
	hash := seq.GetTxHash()
	return &hash
}

func (dag *Dag) GetSequencer(hash types.Hash, seqId uint64) *types.Sequencer {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *types.Sequencer:
		if tx.Id != seqId {
			log.Warn("seq id mismatch ")
			return nil
		}
		return tx
	default:
		return nil
	}
}

func (dag *Dag)	GetConfirmTime(seqId uint64)*types.ConfirmTime {
	dag.mu.RLock()
	defer dag.mu.RUnlock()
	return dag.getConfirmTime(seqId)
}

func (dag*Dag)getConfirmTime(seqId uint64)*types.ConfirmTime {
	if seqId ==0 {
		return nil
	}
	cf:= dag.accessor.readConfirmTime(seqId)
	if cf == nil {
		log.Warn("ConfirmTime not found")
	}
	return cf
}

func (dag *Dag) GetTxsHashesByNumber(id uint64) *types.Hashs {
	dag.mu.RLock()
	defer dag.mu.RUnlock()

	return dag.getTxsHashesByNumber(id)
}

func (dag *Dag) getTxsHashesByNumber(id uint64) *types.Hashs {
	if id > dag.latestSeqencer.Number() {
		return nil
	}
	hashs, err := dag.accessor.ReadIndexedTxHashs(id)
	if err != nil {
		log.Warn("head not found")
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
	return dag.statedb.GetNonce(addr)
}

// // HasLatestNonce returns true if addr already sent any txs to db.
// func (dag *Dag) HasLatestNonce(addr types.Address) (bool, error) {
//      dag.mu.RLock()
//      defer dag.mu.RUnlock()

//      return dag.hasLatestNonce(addr)
// }

// func (dag *Dag) hasLatestNonce(addr types.Address) (bool, error) {
//      return dag.accessor.HasAddrLatestNonce(addr)
// }

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
	if dag.latestSeqencer.Id+1 != batch.Seq.Id {
		return fmt.Errorf("last sequencer id mismatch old %d, new %d", dag.latestSeqencer.Id, batch.Seq.Id)
	}

	var err error
	dbBatch := dag.db.NewBatch()

	// store the tx and update the state
	for _, batchDetail := range batch.Batch {
		txlist := batchDetail.TxList
		if txlist == nil {
			return fmt.Errorf("batch detail does't have txlist")
		}
		sort.Sort(txlist.keys)
		for _, nonce := range *txlist.keys {
			txi := txlist.get(nonce)
			if txi == nil {
				return fmt.Errorf("can't get tx from txlist, nonce: %d", nonce)
			}

			err = dag.WriteTransaction(dbBatch, txi)
			if err != nil {
				return fmt.Errorf("Write tx into db error: %v", err)
			}

			err = dag.accessor.WriteTxSeqRelation(txi.GetTxHash(), batch.Seq.Id)
			if err != nil {
				return fmt.Errorf("Bound the seq id %d to tx err: %v", batch.Seq.Id, err)
			}
		}
	}
	go dag.statedb.Flush()

	// store the hashs of the txs confirmed by this sequencer.
	txHashNum := 0
	if batch.TxHashes != nil {
		txHashNum = len(*batch.TxHashes)
	}
	if txHashNum > 0 {
		dag.accessor.WriteIndexedTxHashs(batch.Seq.Id, batch.TxHashes)
	}

	// save latest sequencer into db
	err = dag.WriteTransaction(dbBatch, batch.Seq)
	if err != nil {
		return err
	}
	err = dag.accessor.WriteSequencerById(batch.Seq)
	if err != nil {
		return err
	}
	log.Tracef("successfully store seq: %s", batch.Seq.GetTxHash().String())

	// set latest sequencer
	err = dag.accessor.WriteLatestSequencer(batch.Seq)
	if err != nil {
		return err
	}
	dag.latestSeqencer = batch.Seq
	cf := types.ConfirmTime{
		SeqId:batch.Seq.Id,
		TxNum: uint64( txHashNum),
		ConfirmTime:time.Now().Format(time.RFC3339Nano),
	}
	dag.writeConfirmTime(&cf)
	log.Tracef("successfully update latest seq: %s", batch.Seq.GetTxHash().String())
	log.WithField("height", batch.Seq.Id).WithField("txs number ", txHashNum).Info("new height")

	return nil
}
func (dag*Dag)writeConfirmTime (cf *types.ConfirmTime) error  {
	return dag.accessor.writeConfirmTime(cf)
}

func (dag*Dag)ReadConfirmTime(seqId uint64) *types.ConfirmTime {
	return dag.accessor.readConfirmTime(seqId)
}


// WriteTransaction write the tx or sequencer into ogdb. It first writes
// the latest nonce of the tx's sender, then write the ([address, nonce] -> hash)
// relation into db, finally write the tx itself. Data will be overwritten
// if it already exists in db.
func (dag *Dag) WriteTransaction(putter ogdb.Putter, tx types.Txi) error {
	curNonce, err := dag.getLatestNonce(tx.Sender())
	if (err != nil) && (err != types.ErrNonceNotExist) {
		return fmt.Errorf("get latest nonce err: %v", err)
	}

	// Write tx hash. This is aimed to allow users to query tx hash
	// by sender address and tx nonce.
	err = dag.accessor.WriteTxHashByNonce(tx.Sender(), tx.GetNonce(), tx.GetTxHash())
	if err != nil {
		return fmt.Errorf("write latest nonce err: %v", err)
	}

	// Write tx itself
	err = dag.accessor.WriteTransaction(putter, tx)
	if err != nil {
		return err
	}
	if tx.GetType() == types.TxBaseTypeNormal {
		txNormal := tx.(*types.Tx)
		dag.statedb.SubBalance(txNormal.From, txNormal.Value)
		dag.statedb.AddBalance(txNormal.To, txNormal.Value)
	}
	// update the nonce if current nonce is larger than previous, or
	// there is no nonce stored in db.
	if (tx.GetNonce() > curNonce) || (err == types.ErrNonceNotExist) {
		dag.statedb.SetNonce(tx.Sender(), tx.GetNonce())
	}

	return nil
}
