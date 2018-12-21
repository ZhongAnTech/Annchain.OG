package og

import (
	"fmt"
	"time"

	"github.com/annchain/OG/common/crypto"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
)

type Og struct {
	Dag      *core.Dag
	TxPool   *core.TxPool
	Manager  *MessageRouter
	TxBuffer *TxBuffer

	NewLatestSequencerCh chan bool //for broadcasting new latest sequencer to record height

	NetworkId     uint64
	CryptoType    crypto.CryptoType
	quit          chan bool
}

func (og *Og) GetCurrentNodeStatus() StatusData {
	return StatusData{
		CurrentBlock:    og.Dag.LatestSequencer().Hash,
		CurrentId:       og.Dag.LatestSequencer().Id,
		GenesisBlock:    og.Dag.Genesis().Hash,
		NetworkId:       og.NetworkId,
		ProtocolVersion: 999, // this field should not be used
	}
}

type OGConfig struct {
	NetworkId     uint64
	CryptoType    crypto.CryptoType
}

func DefaultOGConfig() OGConfig {
	config := OGConfig{
		NetworkId:     1,
	}
	return config
}

func NewOg(config OGConfig) (*Og, error) {
	og := &Og{
		quit: make(chan bool),
	}

	og.NetworkId = config.NetworkId
	og.CryptoType = config.CryptoType
	db, derr := CreateDB()
	if derr != nil {
		return nil, derr
	}
	olddb, derr := GetOldDb()
	if derr != nil {
		return nil, derr
	}
	dagconfig := core.DagConfig{}
	statedbConfig := state.StateDBConfig{
		PurgeTimer:     time.Duration(viper.GetInt("statedb.purge_timer_s")),
		BeatExpireTime: time.Second * time.Duration(viper.GetInt("statedb.beat_expire_time_s")),
	}
	og.Dag, derr = core.NewDag(dagconfig, statedbConfig, db,olddb)
	if derr != nil {
		return nil, derr
	}

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
	og.TxPool = core.NewTxPool(txpoolconfig, og.Dag)

	// initialize
	if !og.Dag.LoadLastState() {
		// TODO use config to load the genesis
		seq, balance := core.DefaultGenesis(config.CryptoType)
		if err := og.Dag.Init(seq, balance); err != nil {
			return nil, err
		}
	}
	seq := og.Dag.LatestSequencer()
	if seq == nil {
		return nil, fmt.Errorf("dag's latest sequencer is not initialized.")
	}
	og.TxPool.Init(seq)

	// Construct the different synchronisation mechanisms

	//heighter := func() uint64 {
	//	return og.Dag.LatestSequencer().Id
	//}
	//inserter := func(seq *types.Sequencer, txs types.Txs) error {
	//	// If fast sync is running, deny importing weird blocks
	//	//if og.fastSyncMode() {
	//	//	logrus.WithField("number", seq.Number()).WithField("hash", seq.GetTxHash()).Warn("Discarded bad propagated sequencer")
	//	//	return nil
	//	//}
	//	// Mark initial sync done on any fetcher import
	//	og.TxBuffer.AddTxs(seq, txs)
	//	return nil
	//}
	//og.fetcher = fetcher.New(og.GetSequencerByHash, heighter, inserter, og.Manager.Hub.RemovePeer)

	// TODO
	// account manager and protocol manager

	return og, nil
}

func (og *Og) Start() {
	og.Dag.Start()
	og.TxPool.Start()
	//// start sync handlers
	//go og.syncer()
	//go og.txsyncLoop()
	go og.BrodcastLatestSequencer()

	logrus.Info("OG Started")
}

func (og *Og) Stop() {
	// Quit fetcher, txsyncLoop.
	close(og.quit)
	//og.quit <- true

	og.Dag.Stop()
	og.TxPool.Stop()

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

func GetOldDb()(ogdb.Database, error) {
	switch viper.GetString("db.name") {
	case "leveldb":
		path := viper.GetString("leveldb.path")+"old"
		cache := viper.GetInt("leveldb.cache")
		handles := viper.GetInt("leveldb.handles")
		return ogdb.NewLevelDB(path, cache, handles)
	default:
		return ogdb.NewMemDatabase(), nil
	}
}

func (og *Og) GetSequencerByHash(hash types.Hash) *types.Sequencer {
	txi := og.Dag.GetTx(hash)
	switch tx := txi.(type) {
	case *types.Sequencer:
		return tx
	default:
		return nil
	}
}

// TODO: why this?
func (og *Og) BrodcastLatestSequencer() {
	for {
		select {
		case <-og.NewLatestSequencerCh:
			seq := og.Dag.LatestSequencer()
			hash := seq.GetTxHash()
			msg := types.MessageSequencerHeader{Hash: &hash, Number: seq.Number()}
			// latest sequencer updated , broadcast it
			go og.Manager.BroadcastMessage(MessageTypeSequencerHeader, &msg)
		case <-og.quit:
			logrus.Info("hub BroadcastLatestSequencer received quit message. Quitting...")
			return
		}
	}
}
