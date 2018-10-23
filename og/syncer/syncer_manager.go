package syncer

import "time"

type SyncManager struct {
	SyncBuffer ISyncBuffer

	finishInit     bool
	enableSync     bool
	forceSyncCycle uint
	syncFlag       uint32 //1 for is syncing
	// channels for fetcher, syncer, txsyncLoop
	txsyncCh chan *txsync
	quitSync chan struct{}
	// timeouts for channel writing
	timeoutSyncTx *time.Timer
}
