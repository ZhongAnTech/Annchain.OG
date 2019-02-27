package node

import (
	"testing"
	"time"
)

func TestAutoClient_Stop(t *testing.T) {
	a := &AutoClient{}
	a.IntervalMode = IntervalModeRandom
	a.SequencerIntervalUs = 500000
	a.TxIntervalUs = 100000
	a.AutoTxEnabled = false
	a.AutoSequencerEnabled = false
	go a.loop()
	time.Sleep(30 * time.Millisecond)
	a.Resume()
	time.Sleep(3 * time.Second)
	return
}
