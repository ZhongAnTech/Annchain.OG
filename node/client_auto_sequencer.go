package node

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"time"
)

type ClientAutoSequencer struct {
	TxCreator             *og.TxCreator
	TxBuffer              *og.TxBuffer
	BlockTimeMilliSeconds int
	PrivateKey            crypto.PrivateKey
	stop                  bool
	currentID             uint64
	manualChan            chan bool
	TxPool                *core.TxPool
	Dag                   *core.Dag
	SampleAccounts        []account.SampleAccount
}

func (c *ClientAutoSequencer) Init() {
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.SampleAccounts = core.GetSampleAccounts()
}

func (c *ClientAutoSequencer) GenerateRequest() {
	c.currentID++

	addr := c.SampleAccounts[0].Address
	noncePool, errPool := c.TxPool.GetLatestNonce(addr)
	if errPool != nil {
		logrus.WithError(errPool).WithField("addr", addr.String()).Debug("txpool nonce not found")
	}
	nonceDag, errDag := c.Dag.GetLatestNonce(addr)
	if errDag != nil {
		logrus.WithError(errDag).WithField("addr", addr.String()).Warn("dag nonce not found")
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

	seq := c.TxCreator.NewSignedSequencer(addr, c.currentID, []types.Hash{}, nonce, c.PrivateKey)
	if ok := c.TxCreator.SealTx(seq); !ok {
		logrus.Warn("clientAutoSequencer Failed to seal tx")
		return
	}
	logrus.WithField("seq", seq).Infof("sequencer generated")
	// TODO: announce tx
	c.TxBuffer.AddTx(seq)
}

func (c *ClientAutoSequencer) loop() {
	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Millisecond * time.Duration(c.BlockTimeMilliSeconds)).C:
		}
		if c.TxBuffer.Hub.AcceptTxs() {
			c.GenerateRequest()
		} else {
			logrus.Debug("can't generate sequencer when syncing")
		}
	}
}

func (c *ClientAutoSequencer) Start() {
	go c.loop()
}

func (c *ClientAutoSequencer) Stop() {
	c.stop = true
}

func (c *ClientAutoSequencer) ManualSequence() {
	c.manualChan <- true
}

func (ClientAutoSequencer) Name() string {
	return "ClientAutoSequencer"
}
