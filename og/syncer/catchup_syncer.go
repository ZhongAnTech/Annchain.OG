package syncer

import (
	"sync"
	"time"

	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/ffchan"
)

const (
	//forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5 // Amount of peers desired to start syncing

	// This is the target size for the packs of transactions sent by pushPendingTxLoop.
	// A pack can get larger than this if a single transactions exceeds this size.
	txsyncPackSize = 100 * 1024

	// TODO: this value will be set to optimal value in the future.
	// If generating sequencer is very fast with few transactions, it should be bigger,
	// otherwise it should be smaller
	maxBehindHeight = 20
	minBehindHeight = 5
)

type CatchupSyncerStatus int

func (m CatchupSyncerStatus) String() string {
	switch m {
	case Started:
		return "CSSStarted"
	case Stopped:
		return "CSSStopped"
	default:
		return "Unknown"
	}
}

const (
	Started CatchupSyncerStatus = iota
	Stopped
)

type CatchupSyncer struct {
	NodeStatusDataProvider og.NodeStatusDataProvider
	PeerProvider           og.PeerProvider
	Hub                    *og.Hub

	Downloader *downloader.Downloader
	SyncMode   downloader.SyncMode

	// should be enabled until quit
	EnableEvent chan bool
	Enabled     bool

	quitLoopEvent chan bool
	quit          chan bool

	OnWorkingStateChanged         []chan CatchupSyncerStatus
	OnNewTxiReceived              []chan types.Txi
	NewPeerConnectedEventListener chan string
	syncFlag                      bool
	WorkState                     CatchupSyncerStatus
	mu                            sync.RWMutex
}

func (c *CatchupSyncer) Init() {
	c.EnableEvent = make(chan bool)
	c.quitLoopEvent = make(chan bool)
	c.NewPeerConnectedEventListener = make(chan string)
	c.quit = make(chan bool)
}

func (c *CatchupSyncer) Start() {
	go c.eventLoop()
	go c.loopSync()
}

func (c *CatchupSyncer) Stop() {
	<-ffchan.NewTimeoutSender(c.quit, true, "catchupSyncerQuit", 1000).C
	<-ffchan.NewTimeoutSender(c.quitLoopEvent, true, "catchupSyncerQuitLoopEvent", 1000).C
	//c.quit <- true
	//c.quitLoopEvent <- true
}

func (CatchupSyncer) Name() string {
	return "CatchupSyncer"
}

func (c *CatchupSyncer) isUpToDate() bool {
	_, bpHash, seqId, err := c.PeerProvider.BestPeerInfo()
	if err != nil {
		return false
	}
	ourId := c.NodeStatusDataProvider.GetCurrentNodeStatus().CurrentId
	if seqId <= ourId+maxBehindHeight {
		logrus.WithField("bestPeer SeqId", seqId).
			WithField("bestPeerHash", bpHash).
			WithField("our SeqId", ourId).
			Debug("we are now up to date")
		return true
	}
	return false
}

func (c *CatchupSyncer) loopSync() {
	startUp := true
	defer c.Downloader.Terminate()
	for {
		select {
		case <-c.quit:
			logrus.Info("CatchupSyncer loopSync received quit message. Quitting...")
			return
		case peer := <-c.NewPeerConnectedEventListener:
			logrus.WithField("peer", peer).Info("new peer connected")
			if !c.Enabled {
				logrus.Info("catchupSyncer not enabled")
				continue
			}
			if startUp {
				if c.isUpToDate() {
					c.NotifyWorkingStateChanged(Stopped)
					startUp = false
					continue
				}
			}
			go c.syncToLatest()
		case <-time.After(time.Second * 15):
			if !c.Enabled {
				logrus.Info("catchupSyncer not enabled")
				continue
			}
			go c.syncToLatest()
		}

	}
}

func (c *CatchupSyncer) isSyncing() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.syncFlag
}

//getWorkState
func (c *CatchupSyncer) getWorkState() CatchupSyncerStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.WorkState
}

func (c *CatchupSyncer) setSyncFlag() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncFlag = true
}

func (c *CatchupSyncer) unsetSyncFlag() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.syncFlag = false
}

func (c *CatchupSyncer) syncToLatest() error {
	if c.isSyncing() {
		logrus.Info("syncing task is busy")
		return nil
	}
	c.setSyncFlag()
	defer c.unsetSyncFlag()
	didSync := false
	//get best peer ,and sync with this peer until we catchup
	bpId, bpHash, seqId, err := c.PeerProvider.BestPeerInfo()
	if err != nil {
		logrus.WithError(err).Warn("picking up best peer")
		return err
	}
	for {
		ourId := c.NodeStatusDataProvider.GetCurrentNodeStatus().CurrentId

		if seqId < ourId+minBehindHeight {
			break
		}
		if !didSync {
			didSync = true
			if seqId < ourId+maxBehindHeight {
				break
			}
			c.NotifyWorkingStateChanged(Started)
		}
		logrus.WithField("peerId", bpId).WithField("seq", seqId).WithField("ourId", ourId).
			Debug("sync with best peer")

		// Run the sync cycle, and disable fast sync if we've went past the pivot block
		if err := c.Downloader.Synchronise(bpId, bpHash, seqId, c.SyncMode); err != nil {
			logrus.WithError(err).Warn("sync failed")
			return err
		}
		bpHash, seqId, err = c.PeerProvider.GetPeerHead(bpId)
		if err != nil {
			logrus.WithError(err).Warn("sync failed")
			return err
		}
	}
	if c.getWorkState() == Started {
		c.NotifyWorkingStateChanged(Stopped)
	}
	return nil
}

//we don't need to broadcast this ,because we broadcast all of our latest sequencer head when it change
/*
func (c *CatchupSyncer) notifyProgress() {
	nodeStatus := c.NodeStatusDataProvider.GetCurrentNodeStatus()
	if nodeStatus.CurrentId > 0 {
		// We've completed a sync cycle, notify all peers of new state. This path is
		// essential in star-topology networks where a gateway node needs to notify
		// all its out-of-date peers of the availability of a new block. This failure
		// scenario will most often crop up in private and hackathon networks with
		// degenerate connectivity, but it should be healthy for the mainnet too to
		// more reliably update peers or the local TD state.
		//go h.BroadcastBlock(head, false)
		hash := nodeStatus.CurrentBlock
		msg := types.MessageSequencerHeader{Hash: &hash, Number: nodeStatus.CurrentId}
		data, _ := msg.MarshalMsg(nil)
		c.Hub.BroadcastMessage(og.MessageTypeSequencerHeader, data)
	}
}

*/
func (c *CatchupSyncer) eventLoop() {
	for {
		select {
		case v := <-c.EnableEvent:
			logrus.WithField("enable", v).Info("syncer got enable event ")
			c.Enabled = v
		case <-c.quitLoopEvent:
			logrus.Info("syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}

//NotifyWorkingStateChanged if starts status is true ,stops status is  false
func (c *CatchupSyncer) NotifyWorkingStateChanged(status CatchupSyncerStatus) {
	c.WorkState = status
	for _, ch := range c.OnWorkingStateChanged {
		<-ffchan.NewTimeoutSender(ch, status, "NotifyWorkingStateChanged", 1000).C
	}
	c.mu.Lock()
	defer c.mu.Unlock()

}
