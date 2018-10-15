package node

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const (
	IntervalModeConstantInterval = "constant"
	IntervalModeRandom           = "random"
)

type ClientAutoTx struct {
	TxCreator              *og.TxCreator
	TxBuffer               *og.TxBuffer
	TxIntervalMilliSeconds int
	PrivateKey             crypto.PrivateKey
	stop                   bool
	currentID              uint64
	manualChan             chan bool
	TxPool                 *core.TxPool
	Dag                    *core.Dag
	SampleAccounts         []account.SampleAccount
	InstanceCount          int
	mu                     sync.RWMutex
	AccountIds             []int
	IntervalMode           string
}

func (c *ClientAutoTx) Init() {
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.SampleAccounts = core.GetSampleAccounts()
}

func (c *ClientAutoTx) GenerateRequest(from int, to int) {
	c.mu.RLock()
	addr := c.SampleAccounts[from].Address

	noncePool, errPool := c.TxPool.GetLatestNonce(addr)
	if errPool != nil {
		logrus.WithError(errPool).WithField("addr", addr.String()).Error("txpool nonce not found")
	}
	nonceDag, errDag := c.Dag.GetLatestNonce(addr)
	if errDag != nil {
		logrus.WithError(errDag).WithField("addr", addr.String()).Error("dag nonce not found")
	}
	var nonce uint64
	if errPool != nil && errDag != nil {
		nonce = 0
	} else {
		nonce = noncePool
		if noncePool < nonceDag {
			nonce = nonceDag
		}
		nonce++
	}

	tx := c.TxCreator.NewSignedTx(c.SampleAccounts[from].Address, c.SampleAccounts[to].Address,
		math.NewBigInt(0), nonce, c.SampleAccounts[from].PrivateKey)
	c.mu.RUnlock()

	if ok := c.TxCreator.SealTx(tx); !ok {
		logrus.Warn("clientAutoTx failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Infof("tx generated")

	c.mu.Lock()
	c.currentID++
	c.SampleAccounts[from].Nonce++
	c.mu.Unlock()

	// TODO: announce tx
	c.TxBuffer.AddTx(tx)
}

func (c *ClientAutoTx) loop(from int, to int) {
	for !c.stop {
		var sleepDuration time.Duration
		switch c.IntervalMode {
		case IntervalModeConstantInterval:
			sleepDuration = time.Millisecond * time.Duration(c.TxIntervalMilliSeconds)
		case IntervalModeRandom:
			sleepDuration = time.Millisecond * (time.Duration(rand.Intn(c.TxIntervalMilliSeconds-1) + 1))
		default:
			panic(fmt.Sprintf("unkown IntervalMode : %s  ",c.IntervalMode))
		}

		select {
		case <-c.manualChan:
		case <-time.After(sleepDuration):
		}
		if c.TxBuffer.Hub.AcceptTxs() {
			c.GenerateRequest(from, to)
		} else {
			logrus.Debug("can't generate tx when syncing")
		}
	}
}

func (c *ClientAutoTx) Start() {
	for i := 0; i < c.InstanceCount; i++ {
		//a, b := rand.Intn(len(c.SampleAccounts)), rand.Intn(len(c.SampleAccounts))
		a, b := c.AccountIds[i], rand.Intn(len(c.SampleAccounts))
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

func (c *ClientAutoTx) Name() string {
	return "ClientAutoTx"
}
