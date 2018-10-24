package syncer

import (
	"time"

	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/og"
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

type CatchupSyncer struct {
	NodeStatusDataProvider og.NodeStatusDataProvider
	BestPeerProvider       og.BestPeerProvider
	Hub                    *og.Hub

	Downloader *downloader.Downloader
	SyncMode   downloader.SyncMode

	// should be enabled until quit
	EnableEvent chan bool
	enabled     bool

	quitLoopEvent chan bool
	quit          bool

	OnWorkingStateChanged []chan bool
}

func (c *CatchupSyncer) Init() {
	c.EnableEvent = make(chan bool)
	c.quitLoopEvent = make(chan bool)
}

func (c *CatchupSyncer) Start() {
	go c.eventLoop()
	go c.loopSync()
}

func (c *CatchupSyncer) Stop() {
	c.quit = true
	c.quitLoopEvent <- true
}

func (CatchupSyncer) Name() string {
	return "CatchupSyncer"
}

func (c *CatchupSyncer) isUpToDate() bool {
	_, bpHash, seqId, err := c.BestPeerProvider.BestPeerInfo()
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
	didSync := false
	for !c.quit {
		if !c.enabled{
			time.Sleep(time.Second * 5)
			continue
		}

		if c.isUpToDate() {
			if didSync {
				c.notifyProgress()
				didSync = false
			}

			time.Sleep(time.Second * 5)
			continue
		}

		err := c.syncToLatest()
		if err == nil {
			didSync = true
		}
		time.Sleep(time.Second * 5)
	}
}
func (c *CatchupSyncer) syncToLatest() error {
	bpId, bpHash, seqId, err := c.BestPeerProvider.BestPeerInfo()

	if err != nil {
		logrus.WithError(err).Warn("picking up best peer")
		return err
	}

	logrus.WithField("peerId", bpId).WithField("seq", seqId).
		Debug("sync with best peer")

	// Run the sync cycle, and disable fast sync if we've went past the pivot block
	if err := c.Downloader.Synchronise(bpId, bpHash, seqId, c.SyncMode); err != nil {
		logrus.WithError(err).Warn("sync failed")
		return err
	}
	return nil
}
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

func (c *CatchupSyncer) eventLoop() {
	for {
		select {
		case v := <-c.EnableEvent:
			logrus.WithField("enable", v).Info("syncer got enable event ")
			c.enabled = v
		case <-c.quitLoopEvent:
			logrus.Info("syncer eventLoop received quit message. Quitting...")
			return
		}
	}
}
