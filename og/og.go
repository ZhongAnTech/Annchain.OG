package og

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/og/downloader"
	"sync/atomic"
	"github.com/annchain/OG/og/fetcher"
	"time"
	"github.com/annchain/OG/types"
)

type Og struct {
	Dag        *core.Dag
	TxPool     *core.TxPool
	Manager    *MessageRouter
	TxBuffer   *TxBuffer
	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher

	// moved from hub to here.
	fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	OnEnableTxsEvent     []chan bool
	NewLatestSequencerCh chan bool //for broadcasting new latest sequencer to record height

	bootstrapNode bool

	NetworkId uint64
}

func (h *Og) GetCurrentNodeStatus() StatusData {
	return StatusData{
		CurrentBlock:    h.Dag.LatestSequencer().Hash,
		CurrentId:       h.Dag.LatestSequencer().Id,
		GenesisBlock:    h.Dag.Genesis().Hash,
		NetworkId:       h.NetworkId,
		ProtocolVersion: 999, // this field should not be used
	}
}

type OGConfig struct {
	Mode           downloader.SyncMode
	BootstrapNode  bool //start accept txs even if no peers
	EnableSync     bool
	ForceSyncCycle uint //millisecends
	NetworkId      uint64
}

func DefaultOGConfig() OGConfig {
	config := OGConfig{
		EnableSync:     true,
		ForceSyncCycle: 10000,
		BootstrapNode:  false,
		NetworkId:      1,
	}
	return config
}

func NewOg(config OGConfig) (*Og, error) {
	og := &Og{}

	og.enableSync = config.EnableSync
	og.forceSyncCycle = config.ForceSyncCycle
	og.bootstrapNode = config.BootstrapNode
	og.NetworkId = config.NetworkId
	if og.forceSyncCycle == 0 {
		og.forceSyncCycle = 10000
	}

	og.txsyncCh = make(chan *txsync)
	og.quitSync = make(chan struct{})
	og.timeoutSyncTx = time.NewTimer(time.Second * 10)

	db, derr := CreateDB()
	if derr != nil {
		return nil, derr
	}
	dagconfig := core.DagConfig{}
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
	og.Dag = core.NewDag(dagconfig, db)
	og.TxPool = core.NewTxPool(txpoolconfig, og.Dag)

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
	og.TxPool.Init(seq)

	// Figure out whether to allow fast sync or not
	if config.Mode == downloader.FastSync && og.Dag.LatestSequencer().Id > 0 {
		logrus.Warn("dag not empty, fast sync disabled")
		config.Mode = downloader.FullSync
	}
	if config.Mode == downloader.FastSync {
		og.fastSync = uint32(1)
	}

	og.downloader = downloader.New(config.Mode, og.Dag, og.Manager.Hub.RemovePeer, og.AddTxs)
	// Construct the different synchronisation mechanisms

	heighter := func() uint64 {
		return og.Dag.LatestSequencer().Id
	}
	inserter := func(seq *types.Sequencer, txs types.Txs) error {
		// If fast sync is running, deny importing weird blocks
		if og.fastSyncMode() {
			logrus.WithField("number", seq.Number()).WithField("hash", seq.GetTxHash()).Warn("Discarded bad propagated sequencer")
			return nil
		}
		// Mark initial sync done on any fetcher import
		og.enableAcceptTx()
		//todo fetch will done later
		//log.Warn("maybe some problems here")
		og.TxBuffer.AddTxs(seq, txs)
		return nil
	}
	og.fetcher = fetcher.New(og.GetSequencerByHash, heighter, inserter, og.Manager.Hub.RemovePeer)

	// TODO
	// account manager and protocol manager

	return og, nil
}

func (og *Og) Start() {
	og.Dag.Start()
	og.TxPool.Start()
	go func() {
		// if disabled sync just accept txs
		if og.bootstrapNode || !og.enableSync {
			og.enableAcceptTx()
		} else {
			og.disableAcceptTx()
		}
	}()
	// start sync handlers
	go og.syncer()
	go og.txsyncLoop()
	go og.BrodcastLatestSequencer()

	logrus.Info("OG Started")
}
func (og *Og) Stop() {
	// Quit fetcher, txsyncLoop.
	close(h.quitSync)
	h.quit <- true

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

func (h *Og) AcceptTxs() bool {
	if atomic.LoadUint32(&h.acceptTxs) == 1 {
		return true
	}
	return false
}

func (h *Og) enableAcceptTx() {
	atomic.StoreUint32(&h.acceptTxs, 1)
	for _, c := range h.OnEnableTxsEvent {
		c <- true
	}
	logrus.Warn("enable accept txs")
}

func (h *Og) disableAcceptTx() {
	atomic.StoreUint32(&h.acceptTxs, 0)
	for _, c := range h.OnEnableTxsEvent {
		c <- false
	}
	logrus.Warn("disable accept txs")
}

func (h *Og) fastSyncMode() bool {
	if atomic.LoadUint32(&h.fastSync) == 1 {
		return true
	}
	return false
}

func (h *Og) isSyncing() bool {
	if atomic.LoadUint32(&h.syncFlag) == 1 {
		return true
	}
	return false
}

func (h *Og) setSyncFlag() {
	atomic.StoreUint32(&h.syncFlag, 1)
}

func (h *Og) unsetSyncFlag() {
	atomic.StoreUint32(&h.syncFlag, 0)
}

func (h *Og) disableFastSync() {
	atomic.StoreUint32(&h.fastSync, 0)
}

func (h *Og) AddTxs(txs types.Txs, seq *types.Sequencer) error {
	var txis []types.Txi
	for _, tx := range txs {
		t := *tx
		txis = append(txis, &t)
	}
	if seq == nil {
		err := fmt.Errorf("seq is nil")
		logrus.WithError(err)
		return err
	}
	if seq.Id != h.Dag.LatestSequencer().Id+1 {
		logrus.WithField("latests seq id ", h.Dag.LatestSequencer().Id).WithField("seq id", seq.Id).Warn("id mismatch")
		return nil
	}
	se := *seq

	return h.SyncBuffer.AddTxs(txis, &se)
}

func (h *Og) GetSequencerByHash(hash types.Hash) *types.Sequencer {
	txi := h.Dag.GetTx(hash)
	switch tx := txi.(type) {
	case *types.Sequencer:
		return tx
	default:
		return nil
	}
}

func (h *Og) BrodcastLatestSequencer() {
	for {
		select {
		case <-h.NewLatestSequencerCh:
			seq := h.Dag.LatestSequencer()
			hash := seq.GetTxHash()
			msgTx := types.MessageSequencerHeader{Hash: &hash, Number: seq.Number()}
			data, _ := msgTx.MarshalMsg(nil)
			// latest sequencer updated , broadcast it
			go h.Manager.BroadcastMessage(MessageTypeSequencerHeader, data)
		case <-h.quit:
			logrus.Info("hub BrodcastLatestSequencer reeived quit message. Quitting...")
			return
		}
	}
}
