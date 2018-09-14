package miner

import (
	"testing"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"github.com/magiconair/properties/assert"
	"time"
)

func TestPoW(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	t.Parallel()

	tx := types.SampleTx()
	miner := PoWMiner{}

	responseChan := make(chan uint64)
	start := time.Now()
	go miner.StartMine(tx, types.HexToHash("0x00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 0, responseChan)

	c, ok := <-responseChan
	logrus.Infof("time: %d ms", time.Since(start).Nanoseconds()/1000000)
	assert.Equal(t, ok, true)
	logrus.Info(c)

}
