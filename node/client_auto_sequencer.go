package node

import (
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/common/crypto"
	"time"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type ClientAutoSequencer struct {
	TxCreator        *og.TxCreator
	TxBuffer         *og.TxBuffer
	BlockTimeSeconds int
	PrivateKey       crypto.PrivateKey
	stop             bool
	currentNonce     uint64
	currentID        uint64
	manualChan       chan bool
}

func (c *ClientAutoSequencer) loop() {
	for !c.stop {
		select {
		case <-c.manualChan:
		case <-time.NewTimer(time.Second * time.Duration(c.BlockTimeSeconds)).C:
		}

		seq := c.TxCreator.NewSignedSequencer(c.currentID, []types.Hash{}, c.currentNonce, c.PrivateKey)
		if ok := c.TxCreator.SealTx(seq); !ok {
			logrus.Warn("ClientAutoSequencer Failed to seal tx")
			continue
		}
		logrus.Infof("Sequencer generated: %s", seq.GetTxHash().Hex())
		logrus.Infof("%+v", seq)
		// TODO: announce tx
		c.TxBuffer.AddTx(seq)
	}
}

func (c *ClientAutoSequencer) Start() {
	c.stop = false
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
