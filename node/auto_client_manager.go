package node

import (
	"github.com/annchain/OG/account"
	"time"
	"sync"
	"github.com/spf13/viper"
	"github.com/annchain/OG/core"
)

type AutoClientManager struct {
	Clients        []*AutoClient
	SampleAccounts []account.SampleAccount
	EnableEvent    chan bool
	stop           bool
	stopped        sync.Cond
}

func (m *AutoClientManager) Init(accountIndices []int, delegate *Delegate) {
	m.Clients = []*AutoClient{}
	m.EnableEvent = make(chan bool)
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
			AutoSequencerEnabled: viper.GetBool("auto_client.sequencer.enabled") && sequencers > 0,
		}
		m.Clients = append(m.Clients, client)
		sequencers --
	}
}

func (m *AutoClientManager) Start() {
	for _, client := range m.Clients {
		client.Start()
	}

	go m.evevtLoop()
}

func (m *AutoClientManager) Stop() {
	m.stop = true
	m.stopped.Wait()
}

func (m *AutoClientManager) Name() string {
	return "AutoClientManager"
}

func (c *AutoClientManager) evevtLoop() {
	defer c.stopped.Signal()
	for !c.stop {
		select {
		case v := <-c.EnableEvent:
			for _, client := range c.Clients {
				if !v {
					client.Pause()
				} else {
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
	}
}
