package core

import (
	"fmt"
	// "fmt"
	"sync"

	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
)

type DagConfig struct{}

type Dag struct {
	conf DagConfig

	db       ogdb.Database
	accessor *Accessor

	genesis        *types.Sequencer
	latestSeqencer *types.Sequencer

	close chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex
}

func NewDag(conf DagConfig, db ogdb.Database) *Dag {
	dag := &Dag{
		conf:     conf,
		db:       db,
		accessor: NewAccessor(db),
		close:    make(chan struct{}),
	}

	return dag
}

type ConfirmBatch struct {
	Seq      *types.Sequencer
	Batch    map[types.Address]*BatchDetail
	TxHashes *types.Hashs
}

// BatchDetail describes all the details of a specific address within a
// sequencer confirmation term.
// - txList - represents the txs sent by this addrs.
// - neg - means the amount this address should spent out.
// - pos - means the amount this address get paid.
type BatchDetail struct {
	TxList map[types.Hash]types.Txi
	Neg    *math.BigInt
	Pos    *math.BigInt
}

func (dag *Dag) Start() {
	log.Infof("TxPool Start")

	// go dag.loop()
}

func (dag *Dag) Stop() {
	close(dag.close)
	dag.wg.Wait()
	log.Infof("TxPool Stopped")
}

// Init inits genesis sequencer and genesis state of the network.
func (dag *Dag) Init(genesis *types.Sequencer, genesisBalance map[types.Address]*math.BigInt) error {
	if genesis.Id != 0 {
		return fmt.Errorf("invalid genesis: id is not zero")
	}
	var err error
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
	err = dag.accessor.WriteTransaction(genesis)
	if err != nil {
		return err
	}
	log.Debugf("successfully store genesis: %s", genesis.String())

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

// LoadLastState load genesis and latestsequencer  data from ogdb. return false if there
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
	return dag.push(batch)
}

// GetTx gets tx from dag network indexed by tx hash. This function querys
// ogdb only.
func (dag *Dag) GetTx(hash types.Hash) types.Txi {
	return dag.getTx(hash)
}

func (dag *Dag) GetSequencerByHash(hash types.Hash) *types.Sequencer {
	tx := dag.getTx(hash)
	switch tx := tx.(type) {
	case *types.Sequencer:
		return tx
	default:
		return nil
	}
}

func (dag *Dag) GetSequencerById(id uint64) *types.Sequencer {
	return dag.getSequencerById(id)
}

func (dag *Dag) getSequencerById(id uint64) *types.Sequencer {
	if id == 0 {
		return dag.genesis
	}
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()
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
	return dag.getSequencerHashById(id)
}

func (dag *Dag) GetSequencer(hash types.Hash, seqId uint64) *types.Sequencer {
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

// GetBalance read the confirmed balance of an address from ogdb.
func (dag *Dag) GetBalance(addr types.Address) *math.BigInt {
	return dag.accessor.ReadBalance(addr)
}

// RollBack rolls back the dag network.
func (dag *Dag) RollBack() {
	// TODO
}

func (dag *Dag) push(batch *ConfirmBatch) error {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	if dag.latestSeqencer.Id+1 != batch.Seq.Id {
		err := fmt.Errorf("last sequencer id mismatch old %d, new %d", dag.latestSeqencer.Id, batch.Seq.Id)
		log.Error(err)
		panic(err)
	}
	// store the tx and update the state
	var err error
	for _, batchDetail := range batch.Batch {
		for _, txi := range batchDetail.TxList {
			err = dag.accessor.WriteTransaction(txi)
			if err != nil {
				return fmt.Errorf("Write tx into db error: %v", err)
			}
			switch tx := txi.(type) {
			case *types.Sequencer:
				break
			case *types.Tx:
				// TODO handle the db error
				dag.accessor.SubBalance(tx.From, tx.Value)
				dag.accessor.AddBalance(tx.To, tx.Value)
				break
			}
		}
	}
	// store the hashs of the txs confirmed by this sequencer.
	txHashNum := 0
	if batch.TxHashes != nil {
		txHashNum = len(*batch.TxHashes)
	}
	if txHashNum > 0 {
		dag.accessor.WriteIndexedTxHashs(batch.Seq.Id, batch.TxHashes)
	}

	// save latest sequencer into db
	err = dag.accessor.WriteTransaction(batch.Seq)
	if err != nil {
		return err
	}
	err = dag.accessor.WriteSequencerById(batch.Seq)
	if err != nil {
		return err
	}
	log.Debugf("successfully store seq: %s", batch.Seq.GetTxHash().String())

	// set latest sequencer
	err = dag.accessor.WriteLatestSequencer(batch.Seq)
	if err != nil {
		return err
	}
	dag.latestSeqencer = batch.Seq

	log.Debugf("successfully update latest seq: %s", batch.Seq.GetTxHash().String())
	log.WithField("height", batch.Seq.Id).WithField("txs number ", txHashNum).Info("new height")

	return nil
}

func (dag *Dag) GetTxsHashesByNumber(id uint64) *types.Hashs {
	return dag.getTxsHashesByNumber(id)
}

func (dag *Dag) getSequencerHashById(id uint64) *types.Hash {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	seq, err := dag.accessor.ReadSequencerById(id)
	if err != nil || seq == nil {
		log.WithField("id", id).Warn("head not found")
		return nil
	}
	hash := seq.GetTxHash()
	return &hash
}

func (dag *Dag) getTxsHashesByNumber(id uint64) *types.Hashs {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()
	if id > dag.latestSeqencer.Number() {
		return nil
	}
	hashs, err := dag.accessor.ReadIndexedTxHashs(id)
	if err != nil {
		log.Warn("head not found")
	}
	return hashs
}

func (dag *Dag) getTx(hash types.Hash) types.Txi {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	return dag.accessor.ReadTransaction(hash)
}

func (dag *Dag) GetTxsByNumber(id uint64) []*types.Tx {
	hashs := dag.GetTxsHashesByNumber(id)
	if hashs == nil {
		return nil
	}
	if len(*hashs) == 0 {
		return nil
	}
	log.WithField("len tx ", len(*hashs)).WithField("id", id).Info("get txs")
	return dag.GetTxs(*hashs)
}

func (dag *Dag) GetTxs(hashs []types.Hash) []*types.Tx {
	return dag.getTxs(hashs)
}

func (dag *Dag) getTxs(hashs []types.Hash) []*types.Tx {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()
	var txs []*types.Tx
	for _, hash := range hashs {
		tx := dag.accessor.ReadTransaction(hash)
		switch tx := tx.(type) {
		case *types.Tx:
			txs = append(txs, tx)
		}
	}
	return txs
}

// func (dag *Dag) loop() {

// 	for {
// 		select {
// 		// TODO

// 		default:
// 			break
// 		}
// 	}
// }
