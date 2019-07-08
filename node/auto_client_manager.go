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
package node

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/goroutine"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"sync"
)

type AutoClientManager struct {
	Clients                []*AutoClient
	SampleAccounts         []*account.SampleAccount
	UpToDateEventListener  chan bool
	NodeStatusDataProvider og.NodeStatusDataProvider
	quit                   chan bool
	wg                     sync.WaitGroup
	RegisterReceiver       func(c chan types.Txi)
	delegate               *Delegate
}

func (m *AutoClientManager) Init(accountIndices []int, delegate *Delegate, coinBaseAccount *account.SampleAccount) {
	m.Clients = []*AutoClient{}
	m.UpToDateEventListener = make(chan bool)
	m.quit = make(chan bool)
	m.delegate = delegate
	// to make sure we have only one sequencer
	sequencers := 1
	mode := viper.GetString("mode")
	var tpsTest bool
	if mode == "tps_test" {
		tpsTest = true
		accountIndices = []int{0}
	}
	for _, accountIndex := range accountIndices {
		client := &AutoClient{
			Delegate:             delegate,
			SampleAccounts:       m.SampleAccounts,
			MyIndex:              accountIndex,
			MyAccount:            m.SampleAccounts[accountIndex],
			NonceSelfDiscipline:  viper.GetBool("auto_client.nonce_self_discipline"),
			IntervalMode:         viper.GetString("auto_client.tx.interval_mode"),
			SequencerIntervalUs:  viper.GetInt("auto_client.sequencer.interval_us"),
			TxIntervalUs:         viper.GetInt("auto_client.tx.interval_us"),
			AutoTxEnabled:        viper.GetBool("auto_client.tx.enabled"),
			AutoSequencerEnabled: viper.GetBool("auto_client.sequencer.enabled") && sequencers > 0 && accountIndex == 0,
			AutoArchiveEnabled:   viper.GetBool("auto_client.archive.enabled"),
			ArchiveInterValUs:    viper.GetInt("auto_client.archive.interval_us"),
			TpsTest:              tpsTest,
		}
		client.Init()
		m.Clients = append(m.Clients, client)
		if client.AutoSequencerEnabled {
			sequencers--
		}

	}
	if !tpsTest && sequencers != 0 && viper.GetBool("auto_client.sequencer.enabled") {
		// add pure sequencer
		client := &AutoClient{
			Delegate:             delegate,
			SampleAccounts:       m.SampleAccounts,
			MyAccount:            m.SampleAccounts[0],
			NonceSelfDiscipline:  viper.GetBool("auto_client.nonce_self_discipline"),
			IntervalMode:         viper.GetString("auto_client.tx.interval_mode"),
			SequencerIntervalUs:  viper.GetInt("auto_client.sequencer.interval_us"),
			TxIntervalUs:         viper.GetInt("auto_client.tx.interval_us"),
			AutoTxEnabled:        false, // always false. If a sequencer is also a tx maker, it will be already added above
			AutoSequencerEnabled: true,
		}
		client.Init()
		m.Clients = append(m.Clients, client)
	}

	if !tpsTest && viper.GetBool("annsensus.campaign") {
		// add pure sequencer
		client := &AutoClient{
			Delegate:             delegate,
			SampleAccounts:       m.SampleAccounts,
			MyAccount:            coinBaseAccount,
			NonceSelfDiscipline:  viper.GetBool("auto_client.nonce_self_discipline"),
			IntervalMode:         viper.GetString("auto_client.tx.interval_mode"),
			SequencerIntervalUs:  viper.GetInt("auto_client.sequencer.interval_us"),
			TxIntervalUs:         viper.GetInt("auto_client.tx.interval_us"),
			AutoTxEnabled:        false, // always false. If a sequencer is also a tx maker, it will be already added above
			AutoSequencerEnabled: false,
			CampainEnable:        true,
		}
		client.Init()
		m.RegisterReceiver(client.NewRawTx)
		m.Clients = append(m.Clients, client)
	}
}

func (m *AutoClientManager) SetTxIntervalUs(interval int) {
	for i := 0; i < len(m.Clients); i++ {
		if m.Clients[i] != nil {
			m.Clients[i].SetTxIntervalUs(interval)
		}

	}
}

func (m *AutoClientManager) Start() {
	for _, client := range m.Clients {
		m.wg.Add(1)
		client.Start()
	}
	m.wg.Add(1)
	goroutine.New(m.eventLoop)
}

func (m *AutoClientManager) Stop() {
	close(m.quit)
	m.wg.Wait()
}

func (m *AutoClientManager) Name() string {
	return "AutoClientManager"
}

func (c *AutoClientManager) eventLoop() {
	defer c.wg.Done()
	for {
		select {
		case v := <-c.UpToDateEventListener:
			for _, client := range c.Clients {
				if !v {
					logrus.Info("pausing client")
					client.Pause()
				} else {
					logrus.Info("resuming client")
					client.Resume()
				}
			}
		//case <-time.After(time.Second * 20):
		//continue
		case <-c.quit:
			logrus.Info("got quit signal")
			// wait for all clients to stop
			for _, client := range c.Clients {
				client.Stop()
				c.wg.Done()
			}
			return
		}

	}

}

func (a *AutoClientManager) JudgeNonce(me *account.SampleAccount) uint64 {

	var n uint64
	//NonceSelfDiscipline
	// fetch from db every time
	n, err := a.delegate.GetLatestAccountNonce(me.Address)
	me.SetNonce(n)
	if err != nil {
		// not exists, set to 0
		return 0
	} else {
		n, _ = me.ConsumeNonce()
		return n
	}
}
