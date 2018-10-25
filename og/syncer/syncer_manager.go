package syncer

import (
	"github.com/annchain/OG/og/downloader"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/og"
)

type SyncManagerConfig struct {
	Mode           downloader.SyncMode
	EnableSync     bool
	ForceSyncCycle uint //millisecends
}

type SyncStatus int

const (
	SyncStatusIncremental SyncStatus = iota
	SyncStatusFull
)

type SyncManager struct {
	Hub                    *og.Hub
	CatchupSyncer          *CatchupSyncer
	IncrementalSyncer      *IncrementalSyncer
	NodeStatusDataProvider og.NodeStatusDataProvider

	OnEnableTxs []chan bool // listeners registered for enabling/disabling generating and receiving txs (fully synced or not)

	// moved from hub to here.
	//fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	//acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	enableSync bool
	//forceSyncCycle uint
	//syncFlag       uint32 //1 for is syncing

	NewPeerConnectedEventListener    chan string
	CatchupSyncerWorkingStateChanged chan bool
	quitFlag                         bool
	Status                           SyncStatus

	//OnNewTxiReceived []chan types.Txi	// for both incremental tx and catchup tx
}

func (h *SyncManager) GetBenchmarks() map[string]interface{} {
	return map[string]interface{}{}
}

func (h *SyncManager) Start() {
	// start full sync listener

	// start incremental sync listener
	// start tx announcement
	// short cut. make it formal later
	//h.CatchupSyncer.OnNewTxiReceived = h.OnNewTxiReceived
	//h.IncrementalSyncer.OnNewTxiReceived = h.OnNewTxiReceived
	h.CatchupSyncer.OnWorkingStateChanged = append(h.CatchupSyncer.OnWorkingStateChanged, h.CatchupSyncerWorkingStateChanged)

	h.CatchupSyncer.Start()
	h.IncrementalSyncer.Start()
	go h.loopSync()
}

func (h *SyncManager) Stop() {
	h.quitFlag = true
}

func (h *SyncManager) Name() string {
	return "SyncManager"
}

func NewSyncManager(config SyncManagerConfig, hub *og.Hub, NodeStatusDataProvider og.NodeStatusDataProvider) *SyncManager {
	sm := &SyncManager{
		Hub:                              hub,
		NodeStatusDataProvider:           NodeStatusDataProvider,
		NewPeerConnectedEventListener:    make(chan string),
		CatchupSyncerWorkingStateChanged: make(chan bool),
	}

	// Figure out whether to allow fast sync or not
	if config.Mode == downloader.FastSync && sm.NodeStatusDataProvider.GetCurrentNodeStatus().CurrentId > 0 {
		logrus.Warn("dag not empty, fast sync disabled")
		config.Mode = downloader.FullSync
	}
	if config.Mode == downloader.FastSync {
		//sm.fastSync = uint32(1)
	}

	sm.enableSync = config.EnableSync
	//sm.forceSyncCycle = config.ForceSyncCycle
	return sm
}

// loopSync constantly check if there is new peer connected
func (h *SyncManager) loopSync() {
	h.CatchupSyncer.EnableEvent <- true
	h.IncrementalSyncer.EnableEvent <- false
	h.Status = SyncStatusFull

	for !h.quitFlag {
		// listen to either full sync or incremental sync to get something new.
		select {
		case peer := <-h.NewPeerConnectedEventListener:
			logrus.WithField("peer", peer).Info("new peer connected")
		case status := <-h.CatchupSyncerWorkingStateChanged:
			logrus.WithField("v", status).Info("catchup syncer working state changed")
			h.IncrementalSyncer.EnableEvent <- status
			if status {
				h.Status = SyncStatusIncremental
			} else {
				h.Status = SyncStatusFull
			}
			for _, c := range h.OnEnableTxs {
				c <- status
			}
		}
	}
}
