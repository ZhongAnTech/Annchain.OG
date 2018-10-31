package og

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/ogdb"
)

type Og struct {
	Dag     *core.Dag
	Txpool  *core.TxPool
	Manager *MessageRouter
}

func NewOg() (*Og, error) {
	og := &Og{}

	db, derr := CreateDB()
	if derr != nil {
		return nil, derr
	}
	dagconfig := core.DagConfig{}
	og.Dag = core.NewDag(dagconfig, db)

	txpoolconfig := core.TxPoolConfig{
		QueueSize:              viper.GetInt("txpool.queue_size"),
		TipsSize:               viper.GetInt("txpool.tips_size"),
		ResetDuration:          viper.GetInt("txpool.reset_duration"),
		TxVerifyTime:           viper.GetInt("txpool.tx_verify_time"),
		TxValidTime:            viper.GetInt("txpool.tx_valid_time"),
		TimeOutPoolQueue:       viper.GetInt("txpool.timeout_pool_queue_ms"),
		TimeoutSubscriber:      viper.GetInt("txpool.timeout_subscriber_ms"),
		TimeoutConfirmation:    viper.GetInt("txpool.timeout_confirmation_ms"),
		TimeoutLatestSequencer: viper.GetInt("txpool.timeout_latest_seq_ms"),
	}
	og.Txpool = core.NewTxPool(txpoolconfig, og.Dag)

	// initialize
	if !og.Dag.LoadLastState() {
		// TODO use config to load the genesis
		seq, balance := core.DefaultGenesis()
		if err := og.Dag.Init(seq, balance); err != nil {
			return nil, err
		}
	}
	seq := og.Dag.LatestSequencer()
	if seq == nil {
		return nil, fmt.Errorf("dag's latest sequencer is not initialized.")
	}
	og.Txpool.Init(seq)

	// TODO
	// account manager and protocol manager

	return og, nil
}

func (og *Og) Start() {
	og.Dag.Start()
	og.Txpool.Start()

	logrus.Info("OG Started")
}
func (og *Og) Stop() {
	og.Dag.Stop()
	og.Txpool.Stop()

	logrus.Info("OG Stopped")
}

func (og *Og) Name() string {
	return "OG"
}

func CreateDB() (ogdb.Database, error) {
	switch viper.GetString("db.name") {
	case "leveldb":
		path := viper.GetString("leveldb.path")
		cache := viper.GetInt("leveldb.cache")
		handles := viper.GetInt("leveldb.handles")
		return ogdb.NewLevelDB(path, cache, handles)
	default:
		return ogdb.NewMemDatabase(), nil
	}
}
