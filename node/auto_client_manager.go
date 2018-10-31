package node

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/core"
	"github.com/spf13/viper"
	"sync"
	"time"
	"github.com/sirupsen/logrus"
)

type AutoClientManager struct {
	Clients               []*AutoClient
	SampleAccounts        []account.SampleAccount
	UpToDateEventListener chan bool
	stop                  bool
	wg                    sync.WaitGroup
}

func (m *AutoClientManager) Init(accountIndices []int, delegate *Delegate) {
	m.Clients = []*AutoClient{}
	m.UpToDateEventListener = make(chan bool)
	m.SampleAccounts = core.GetSampleAccounts()

	// to make sure we have only one sequencer
	sequencers := 1
	for _, accountIndex := range accountIndices {
		client := &AutoClient{
			Delegate:             delegate,
			SampleAccounts:       m.SampleAccounts,
			MyAccountIndex:       accountIndex,
			NonceSelfDiscipline:  viper.GetBool("auto_client.nonce_self_discipline"),
			IntervalMode:         viper.GetString("auto_client.tx.interval_mode"),
			SequencerIntervalMs:  viper.GetInt("auto_client.sequencer.interval_ms"),
			TxIntervalMs:         viper.GetInt("auto_client.tx.interval_ms"),
			AutoTxEnabled:        viper.GetBool("auto_client.tx.enabled"),
			AutoSequencerEnabled: viper.GetBool("auto_client.sequencer.enabled") && sequencers > 0 && accountIndex == 0,
		}
		client.Init()
		m.Clients = append(m.Clients, client)
		if client.AutoSequencerEnabled {
			sequencers--
		}

	}
	if sequencers != 0 && viper.GetBool("auto_client.sequencer.enabled") {
		// add pure sequencer
		client := &AutoClient{
			Delegate:             delegate,
			SampleAccounts:       m.SampleAccounts,
			MyAccountIndex:       0,
			NonceSelfDiscipline:  viper.GetBool("auto_client.nonce_self_discipline"),
			IntervalMode:         viper.GetString("auto_client.tx.interval_mode"),
			SequencerIntervalMs:  viper.GetInt("auto_client.sequencer.interval_ms"),
			TxIntervalMs:         viper.GetInt("auto_client.tx.interval_ms"),
			AutoTxEnabled:        viper.GetBool("auto_client.tx.enabled"),
			AutoSequencerEnabled: true,
		}
		client.Init()
		m.Clients = append(m.Clients, client)
	}
}

func (m *AutoClientManager) Start() {
	for _, client := range m.Clients {
		m.wg.Add(1)
		client.Start()
	}
	m.wg.Add(1)
	go m.eventLoop()
}

func (m *AutoClientManager) Stop() {
	m.stop = true
	m.wg.Wait()
}

func (m *AutoClientManager) Name() string {
	return "AutoClientManager"
}

func (c *AutoClientManager) eventLoop() {
	defer c.wg.Done()
	for !c.stop {
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
		case <-time.After(time.Second):
			continue
		}
	}
	// wait for all clients to stop
	for _, client := range c.Clients {
		client.Stop()
		c.wg.Done()
	}
}
