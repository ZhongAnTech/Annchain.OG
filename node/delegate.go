package node

import (
	"fmt"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type TxRequest struct {
	AddrFrom   types.Address
	AddrTo     types.Address
	PrivateKey crypto.PrivateKey
	Value      *math.BigInt
	Nonce      uint64
}

type Delegate struct {
	TxCreator         *og.TxCreator
	TxBuffer          *og.TxBuffer
	TxPool            *core.TxPool
	Dag               *core.Dag
	OnNewTxiGenerated []chan types.Txi
}

func (c *Delegate) GenerateTx(r TxRequest) (tx types.Txi, err error) {
	tx = c.TxCreator.NewSignedTx(r.AddrFrom, r.AddrTo, r.Value, r.Nonce, r.PrivateKey)

	if ok := c.TxCreator.SealTx(tx); !ok {
		logrus.Warn("delegate failed to seal tx")
		err = fmt.Errorf("delegate failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	return
}

type SeqRequest struct {
	Issuer     types.Address
	PrivateKey crypto.PrivateKey
	Nonce      uint64
	Height     uint64
}

func (c *Delegate) GenerateSequencer(r SeqRequest) (seq types.Txi, err error) {
	seq = c.TxCreator.NewSignedSequencer(r.Issuer, r.Height, r.Nonce, r.PrivateKey)
	logrus.WithField("seq", seq).Infof("sequencer generated")
	if ok := c.TxCreator.SealTx(seq); !ok {
		logrus.Warn("delegate failed to seal seq")
		err = fmt.Errorf("delegate failed to seal seq")
		return
	}
	logrus.WithField("seq", seq).Infof("sequencer  connected")
	return
}

func (c *Delegate) GetLatestAccountNonce(addr types.Address) (uint64, error) {
	noncePool, errPool := c.TxPool.GetLatestNonce(addr)
	if errPool == nil {
		return noncePool, errPool
	}
	logrus.WithError(errPool).WithField("addr", addr.String()).Trace("txpool nonce not found")

	nonceDag, errDag := c.Dag.GetLatestNonce(addr)
	if errDag == nil {
		return nonceDag, errDag
	}
	logrus.WithError(errDag).WithField("addr", addr.String()).Trace("dag nonce not found")

	return 0, fmt.Errorf("nonce for address not found")
}

func (c *Delegate) GetLatestDagSequencer() *types.Sequencer {
	latestSeq := c.Dag.LatestSequencer()
	return latestSeq
}

func (c *Delegate) Announce(txi types.Txi) {
	for _, ch := range c.OnNewTxiGenerated {
		ch <- txi
		// <-ffchan.NewTimeoutSenderShort(ch, txi, fmt.Sprintf("OnNewTxiGenerated_%d", i)).C
	}
}
