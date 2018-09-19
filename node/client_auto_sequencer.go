package node

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"time"
	"github.com/annchain/OG/core"
)

type ClientAutoSequencer struct {
	TxCreator             *og.TxCreator
	TxBuffer              *og.TxBuffer
	BlockTimeMilliSeconds int
	PrivateKey            crypto.PrivateKey
	stop                  bool
	currentNonce          uint64
	currentID             uint64
	manualChan            chan bool
	Dag                   *core.Dag
}

func (c *ClientAutoSequencer) Init(){
	c.stop = false
	lseq := c.Dag.LatestSequencer()
	if lseq!=nil {
		c.currentID = lseq.Id
	}
}

func (c *ClientAutoSequencer) GenerateRequest(){
	c.currentID ++
	seq := c.TxCreator.NewSignedSequencer(c.currentID, []types.Hash{}, c.currentNonce, c.PrivateKey)
	if ok := c.TxCreator.SealTx(seq); !ok {
		logrus.Warn("clientAutoSequencer Failed to seal tx")
		return
	}
	logrus.WithField("seq", seq).Infof("sequencer generated")
	c.currentNonce ++
	// TODO: announce tx
	c.TxBuffer.AddTx(seq)
}

func (c *ClientAutoSequencer) loop() {
	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Millisecond * time.Duration(c.BlockTimeMilliSeconds)).C:
		}
		c.GenerateRequest()
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
