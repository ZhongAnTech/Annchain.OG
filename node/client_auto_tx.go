package node

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/sirupsen/logrus"
	"time"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/math"
	"math/rand"
)

type ClientAutoTx struct {
	TxCreator         *og.TxCreator
	TxBuffer          *og.TxBuffer
	TxIntervalSeconds int
	PrivateKey        crypto.PrivateKey
	stop              bool
	currentNonce      uint64
	currentID         uint64
	manualChan        chan bool
	Dag               *core.Dag
	sampleAccounts    []account.SampleAccount
	InstanceCount     int
}

func (c *ClientAutoTx) loop(from int, to int, nonce uint64) {

	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Millisecond * time.Duration(rand.Intn(c.TxIntervalSeconds * 1000))).C:
		}
		c.currentID ++
		tx := c.TxCreator.NewSignedTx(c.sampleAccounts[from].Address, c.sampleAccounts[to].Address, math.NewBigInt(1), nonce, c.sampleAccounts[from].PrivateKey)
		if ok := c.TxCreator.SealTx(tx); !ok {
			logrus.Warn("ClientAutoTx Failed to seal tx")
			continue
		}
		logrus.Infof("Tx generated: %s", tx.GetTxHash().Hex())
		logrus.Infof("%+v", tx)
		// TODO: announce tx
		c.TxBuffer.AddTx(tx)
	}
}

func (c *ClientAutoTx) Start() {
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.sampleAccounts = og.GetSampleAccounts()
	for i := 0; i < c.InstanceCount; i++ {
		a, b := rand.Intn(len(c.sampleAccounts)), rand.Intn(len(c.sampleAccounts))
		go c.loop(a, b, 0)
		logrus.Infof("Start auto Tx maker from %d to %d", a, b)
	}

}

func (c *ClientAutoTx) Stop() {
	c.stop = true
}

func (c *ClientAutoTx) ManualSequence() {
	c.manualChan <- true
}

func (ClientAutoTx) Name() string {
	return "ClientAutoTx"
}
