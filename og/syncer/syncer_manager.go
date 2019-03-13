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
package syncer

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/downloader"
)

type SyncManagerConfig struct {
	Mode           downloader.SyncMode
	BootstrapNode  bool
	ForceSyncCycle uint //milliseconds
}

type SyncStatus int

const (
	SyncStatusIncremental SyncStatus = iota
	SyncStatusFull
)

func (m SyncStatus) String() string {
	switch m {
	case SyncStatusIncremental:
		return "SyncStatusIncremental"
	case SyncStatusFull:
		return "SyncStatusFull"
	default:
		return "Default"
	}
}

type SyncManager struct {
	Hub                    *og.Hub
	CatchupSyncer          *CatchupSyncer
	IncrementalSyncer      *IncrementalSyncer
	NodeStatusDataProvider og.NodeStatusDataProvider

	OnUpToDate []chan bool // listeners registered for enabling/disabling generating and receiving txs (fully synced or not)

	// moved from hub to here.
	//fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	//acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	//forceSyncCycle uint
	//syncFlag       uint32 //1 for is syncing
	BootstrapNode                    bool //if bootstrap node just accept txs in starting ,no sync
	CatchupSyncerWorkingStateChanged chan CatchupSyncerStatus
	quitFlag                         bool
	Status                           SyncStatus

	//OnNewTxiReceived []chan types.Txi	// for both incremental tx and catchup tx
}

func (s *SyncManager) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{}
}

func (s *SyncManager) Start() {
	// start full sync listener

	// start incremental sync listener
	// start tx announcement
	// short cut. make it formal later
	//s.CatchupSyncer.OnNewTxiReceived = s.OnNewTxiReceived
	//s.IncrementalSyncer.OnNewTxiReceived = s.OnNewTxiReceived
	s.CatchupSyncer.OnWorkingStateChanged = append(s.CatchupSyncer.OnWorkingStateChanged, s.CatchupSyncerWorkingStateChanged)

	s.CatchupSyncer.Start()
	s.IncrementalSyncer.Start()
	go s.loopSync()
	go func() {
		// if BootstrapNode  just accept txs
		if s.BootstrapNode {
			s.CatchupSyncer.NotifyWorkingStateChanged(Stopped)
		} else {
			s.NotifyUpToDateEvent(false)
		}
	}()
}

func (s *SyncManager) Stop() {
	s.quitFlag = true
}

func (s *SyncManager) Name() string {
	return "SyncManager"
}

func NewSyncManager(config SyncManagerConfig, hub *og.Hub, NodeStatusDataProvider og.NodeStatusDataProvider) *SyncManager {
	sm := &SyncManager{
		Hub:                              hub,
		NodeStatusDataProvider:           NodeStatusDataProvider,
		CatchupSyncerWorkingStateChanged: make(chan CatchupSyncerStatus),
		BootstrapNode:                    config.BootstrapNode,
	}

	// Figure out whether to allow fast sync or not
	if config.Mode == downloader.FastSync && sm.NodeStatusDataProvider.GetCurrentNodeStatus().CurrentId > 0 {
		log.Warn("dag not empty, fast sync disabled")
		config.Mode = downloader.FullSync
	}
	if config.Mode == downloader.FastSync {
		//sm.fastSync = uint32(1)
	}
	//sm.forceSyncCycle = config.ForceSyncCycle
	return sm
}

// loopSync constantly check if there is new peer connected
func (s *SyncManager) loopSync() {
	s.IncrementalSyncer.EnableEvent <- false
	s.CatchupSyncer.EnableEvent <- true
	// <-ffchan.NewTimeoutSender(s.IncrementalSyncer.EnableEvent, false, "timeoutAcquireTx", 1000).C
	// <-ffchan.NewTimeoutSender(s.CatchupSyncer.EnableEvent, true, "timeoutAcquireTx", 1000).C

	s.Status = SyncStatusFull

	for !s.quitFlag {
		// listen to either full sync or incremental sync to get something new.
		select {
		case status := <-s.CatchupSyncerWorkingStateChanged:
			log.WithField("v", status.String()).Info("catchup syncer working state changed")
			switch status {
			case Started:
				// catch up started. pause incremental
				s.Status = SyncStatusFull
				s.IncrementalSyncer.EnableEvent <- false
				// <-ffchan.NewTimeoutSender(s.IncrementalSyncer.EnableEvent, false, "IncrementalSyncerEnable", 1000).C
				s.NotifyUpToDateEvent(false)
			case Stopped:
				// catch up already done. now it is up to date. start incremental
				s.Status = SyncStatusIncremental
				s.IncrementalSyncer.EnableEvent <- true
				// <-ffchan.NewTimeoutSender(s.IncrementalSyncer.EnableEvent, true, "IncrementalSyncerEnable", 1000).C
				s.NotifyUpToDateEvent(true)
			}
		}
	}
}

func (s *SyncManager) NotifyUpToDateEvent(isUpToDate bool) {
	for _, c := range s.OnUpToDate {
		c <- isUpToDate
	}
}
