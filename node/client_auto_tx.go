package node

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/sirupsen/logrus"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/math"
	"math/rand"
	"sync"
	"time"
)

type ClientAutoTx struct {
	TxCreator              *og.TxCreator
	TxBuffer               *og.TxBuffer
	TxIntervalMilliSeconds int
	PrivateKey             crypto.PrivateKey
	stop                   bool
	currentNonce           uint64
	currentID              uint64
	manualChan             chan bool
	Dag                    *core.Dag
	SampleAccounts         []account.SampleAccount
	InstanceCount          int
	mu                     sync.RWMutex
}

func (c *ClientAutoTx) Init() {
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.SampleAccounts = og.GetSampleAccounts()
}

func (c *ClientAutoTx) GenerateRequest(from int, to int) {
	c.mu.RLock()
	tx := c.TxCreator.NewSignedTx(c.SampleAccounts[from].Address, c.SampleAccounts[to].Address, math.NewBigInt(0), c.SampleAccounts[from].Nonce, c.SampleAccounts[from].PrivateKey)
	c.mu.RUnlock()

	if ok := c.TxCreator.SealTx(tx); !ok {
		logrus.Warn("clientAutoTx failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Infof("tx generated")

	c.mu.Lock()
	c.currentID ++
	c.SampleAccounts[from].Nonce ++
	c.mu.Unlock()

	// TODO: announce tx
	c.TxBuffer.AddTx(tx)
}

func (c *ClientAutoTx) loop(from int, to int) {
	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Millisecond * (time.Duration(rand.Intn(c.TxIntervalMilliSeconds) + 1))).C:
		}
		c.GenerateRequest(from, to)
	}
}

func (c *ClientAutoTx) Start() {
	for i := 0; i < c.InstanceCount; i++ {
		a, b := rand.Intn(len(c.SampleAccounts)), rand.Intn(len(c.SampleAccounts))
		go c.loop(a, b)
		logrus.Infof("start auto tx maker from %d to %d", a, b)
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
