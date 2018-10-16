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
	enable                bool
	EnableEvent           chan bool
	quit                  chan bool
}

func (c *ClientAutoSequencer) Init() {
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq != nil {
		c.currentID = lseq.Id
	}
	c.SampleAccounts = core.GetSampleAccounts()
	c.enable = false
	c.EnableEvent = make(chan bool)
	c.quit = make(chan bool)
}

func (c *ClientAutoSequencer) GenerateRequest() {

	addr := c.SampleAccounts[0].Address
	noncePool, errPool := c.TxPool.GetLatestNonce(addr)
	if errPool != nil {
		logrus.WithError(errPool).WithField("addr", addr.String()).Debug("txpool nonce not found")
	}
	nonceDag, errDag := c.Dag.GetLatestNonce(addr)
	if errDag != nil {
		logrus.WithError(errDag).WithField("addr", addr.String()).Debug("dag nonce not found")
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

	latestSeq := c.Dag.LatestSequencer()

	seq := c.TxCreator.NewSignedSequencer(addr, latestSeq.Id+1, []types.Hash{}, nonce, c.PrivateKey)
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
		if c.enable {
			c.GenerateRequest()
		} else {
			logrus.Debug("can't generate sequencer when disabling")
		}
	}
}

func (c *ClientAutoSequencer) Start() {
	go c.evevtLoop()
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

func (c *ClientAutoSequencer) evevtLoop() {
	for !c.stop {
		select {
		case v := <-c.EnableEvent:
			c.enable = v
			logrus.WithField("enable: ", v).Info("got enable event ")
		case <-c.quit:
			logrus.Infof("ClientAutoSequencer received quit signal ,quiting...")
			return
		}
	}
}
