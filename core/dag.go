package core

import (
	"fmt"
	// "fmt"
	"sync"

	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	log "github.com/sirupsen/logrus"
	"github.com/annchain/OG/common/math"
)

type DagConfig struct {}

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
	Seq		*types.Sequencer
	Batch	map[types.Address]*BatchDetail
}
// BatchDetail describes all the details of a specific address within a 
// sequencer confirmation term. 
// - txList - represents the txs sent by this addrs.
// - neg - means the amount this address should spent out. 
// - pos - means the amount this address get paid.
type BatchDetail struct {
	TxList	map[types.Hash]types.Txi
	Neg		*math.BigInt
	Pos		*math.BigInt
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

// LoadGenesis load genesis data from ogdb. return false if there 
// is no genesis stored in the db.
func (dag *Dag) LoadGenesis() bool {
	dag.mu.Lock()
	defer dag.mu.Unlock()

	genesis := dag.accessor.ReadGenesis()
	if genesis == nil {
		return false
	}
	dag.genesis = genesis
	dag.latestSeqencer = genesis
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

	// store the tx and update the state
	var txHashs types.Hashs
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
				txHashs = append(txHashs, tx.GetTxHash())
				// TODO handle the db error
				dag.accessor.SubBalance(tx.From, tx.Value)
				dag.accessor.AddBalance(tx.To, tx.Value)
				break
			}
		}
	}
	// store the hashs of the txs confirmed by this sequencer.
	if len(txHashs) > 0 {
		dag.accessor.WriteIndexedTxHashs(batch.Seq.Id, &txHashs)
	}

	// save latest sequencer into db
	err = dag.accessor.WriteTransaction(batch.Seq)
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

	return nil
}

func (dag *Dag) getTx(hash types.Hash) types.Txi {
	dag.mu.Lock()
	defer dag.mu.Unlock()
	dag.wg.Add(1)
	defer dag.wg.Done()

	return dag.accessor.ReadTransaction(hash)
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
