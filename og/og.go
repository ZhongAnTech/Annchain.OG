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
package og

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/common/io"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/annchain/OG/core"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
)

type Og struct {
	Dag      *core.Dag
	TxPool   *core.TxPool
	Manager  *MessageRouter
	TxBuffer *TxBuffer

	NewLatestSequencerCh chan bool //for broadcasting new latest sequencer to record height

	NetworkId uint64
	quit      chan bool
}

func (og *Og) GetCurrentNodeStatus() p2p_message.StatusData {
	return p2p_message.StatusData{
		CurrentBlock:    og.Dag.LatestSequencer().Hash,
		CurrentId:       og.Dag.LatestSequencer().Height,
		GenesisBlock:    og.Dag.Genesis().Hash,
		NetworkId:       og.NetworkId,
		ProtocolVersion: 999, // this field should not be used
	}
}

func (og *Og) GetHeight() uint64 {
	return og.Dag.LatestSequencer().Height
}

type OGConfig struct {
	NetworkId   uint64
	GenesisPath string
}

func DefaultOGConfig() OGConfig {
	config := OGConfig{
		NetworkId: 1,
	}
	return config
}

func NewOg(config OGConfig) (*Og, error) {
	og := &Og{
		quit:                 make(chan bool),
		NewLatestSequencerCh: make(chan bool),
	}

	og.NetworkId = config.NetworkId
	db, err := CreateDB()
	if err != nil {
		logrus.WithError(err).Warning("create db error")
		return nil, err
	}
	testDb, err := GetOldDb()
	if err != nil {
		return nil, err
	}
	dagConfig := core.DefaultDagConfig()
	dagConfig.GenesisPath = config.GenesisPath
	stateDbConfig := state.StateDBConfig{
		PurgeTimer:     time.Duration(viper.GetInt("statedb.purge_timer_s")),
		BeatExpireTime: time.Second * time.Duration(viper.GetInt("statedb.beat_expire_time_s")),
	}
	og.Dag, err = core.NewDag(dagConfig, stateDbConfig, db, testDb)
	if err != nil {
		logrus.WithError(err).Warning("create db error")
		return nil, err
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

	// // initialize
	// if !og.Dag.LoadLastState() {
	// 	logrus.Trace("no existing data found in db")
	// 	// TODO use config to load the genesis
	// 	seq, balance := core.DefaultGenesis(config.CryptoType)
	// 	if err := og.Dag.Init(seq, balance); err != nil {
	// 		return nil, err
	// 	}
	// }
	seq := og.Dag.LatestSequencer()
	if seq == nil {
		return nil, fmt.Errorf("dag's latest sequencer is not initialized")
	}
	og.TxPool.Init(seq)
	// TODO account manager and protocol manager

	return og, nil
}

func (og *Og) Start() {
	og.Dag.Start()
	og.TxPool.Start()
	//// start sync handlers
	//goroutine.New( og.syncer)
	//goroutine.New(og.txsyncLoop)
	goroutine.New(og.BroadcastLatestSequencer)

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
		path := io.FixPrefixPath(viper.GetString("datadir"), viper.GetString("leveldb.path"))
		cache := viper.GetInt("leveldb.cache")
		handles := viper.GetInt("leveldb.handles")
		return ogdb.NewLevelDB(path, cache, handles)
	default:
		return ogdb.NewMemDatabase(), nil
	}
}

func GetOldDb() (ogdb.Database, error) {
	switch viper.GetString("db.name") {
	case "leveldb":
		path := io.FixPrefixPath(viper.GetString("datadir"), viper.GetString("leveldb.path")+"test")
		cache := viper.GetInt("leveldb.cache")
		handles := viper.GetInt("leveldb.handles")
		return ogdb.NewLevelDB(path, cache, handles)
	default:
		return ogdb.NewMemDatabase(), nil
	}
}

func (og *Og) GetSequencerByHash(hash common.Hash) *tx_types.Sequencer {
	txi := og.Dag.GetTx(hash)
	switch tx := txi.(type) {
	case *tx_types.Sequencer:
		return tx
	default:
		return nil
	}
}

//BroadcastLatestSequencer  broadcast the newest sequencer header , seuqencer header is a network state , representing peer's height
// other peers will know our height and know whether thy were updated and sync with the best height
func (og *Og) BroadcastLatestSequencer() {
	var notSend bool
	var mu sync.RWMutex
	for {
		select {
		case <-og.NewLatestSequencerCh:

			if og.Manager.Hub.Downloader.Synchronising() {
				mu.Lock()
				notSend = true
				mu.Unlock()
				logrus.Debug("sequencer updated, but not broadcasted")
				continue
			}
			logrus.Debug("sequencer updated")
			mu.Lock()
			notSend = false
			mu.Unlock()
			seq := og.Dag.LatestSequencer()
			hash := seq.GetTxHash()
			number := seq.Number()
			msg := p2p_message.MessageSequencerHeader{Hash: &hash, Number: &number}
			// latest sequencer updated , broadcast it
			function := func() {
				og.Manager.BroadcastMessage(p2p_message.MessageTypeSequencerHeader, &msg)
			}
			goroutine.New(function)
		case <-time.After(200 * time.Millisecond):
			if notSend && !og.Manager.Hub.Downloader.Synchronising() {
				mu.Lock()
				notSend = true
				mu.Unlock()
				logrus.Debug("sequencer updated")
				seq := og.Dag.LatestSequencer()
				hash := seq.GetTxHash()
				number := seq.Number()
				msg := p2p_message.MessageSequencerHeader{Hash: &hash, Number: &number}
				// latest sequencer updated , broadcast it
				function := func() {
					og.Manager.BroadcastMessage(p2p_message.MessageTypeSequencerHeader, &msg)
				}
				goroutine.New(function)
			}
		case <-og.quit:
			logrus.Info("hub BroadcastLatestSequencer received quit message. Quitting...")
			return
		}
	}
}
