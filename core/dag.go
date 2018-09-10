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
	}
	return dag
}

type ConfirmBatch struct { 
	seq		*types.Sequencer
	batch	map[types.Address]*BatchDetail
}
// BatchDetail describes all the details of a specific address within a 
// sequencer confirmation term. 
// - txList - represents the txs sent by this addrs.
// - neg - means the amount this address should spent out. 
// - pos - means the amount this address get paid.
type BatchDetail struct {
	txList	map[types.Hash]types.Txi
	neg		*math.BigInt
	pos		*math.BigInt
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

// func (dag *Dag) Status() 

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
	var err error 
	for _, batchDetail := range batch.batch {
		for _, txi := range batchDetail.txList {
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
	
	// set latest sequencer
	err = dag.accessor.WriteLatestSequencer(batch.seq)
	if err != nil {
		return err
	}
	dag.latestSeqencer = batch.seq
	
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
