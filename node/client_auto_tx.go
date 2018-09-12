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
	SampleAccounts    []account.SampleAccount
	InstanceCount     int
}

func (c *ClientAutoTx) Init(){
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.SampleAccounts = og.GetSampleAccounts()
}

func (c *ClientAutoTx) GenerateRequest(from int, to int){
	c.currentID ++
	tx := c.TxCreator.NewSignedTx(c.SampleAccounts[from].Address, c.SampleAccounts[to].Address, math.NewBigInt(1), c.SampleAccounts[from].Nonce, c.SampleAccounts[from].PrivateKey)
	if ok := c.TxCreator.SealTx(tx); !ok {
		logrus.Warn("ClientAutoTx Failed to seal tx")
		return
	}
	logrus.Infof("Tx generated: %s", tx.GetTxHash().Hex())
	logrus.Infof("%+v", tx)
	// TODO: announce tx
	c.TxBuffer.AddTx(tx)
}

func (c *ClientAutoTx) loop(from int, to int) {

	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Millisecond * time.Duration(rand.Intn(c.TxIntervalSeconds * 1000))).C:
		}
		c.GenerateRequest(from, to)
	}
}

func (c *ClientAutoTx) Start() {
	for i := 0; i < c.InstanceCount; i++ {
		a, b := rand.Intn(len(c.SampleAccounts)), rand.Intn(len(c.SampleAccounts))
		go c.loop(a, b)
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
