// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package node

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/annchain/OG/og/txmaker"
	"github.com/sirupsen/logrus"
)

type TxRequest struct {
	AddrFrom   common.Address
	AddrTo     common.Address
	PrivateKey crypto.PrivateKey
	Value      *math.BigInt
	Nonce      uint64
	TokenId    int32
}

type insertTxsFn func(seq *ogmessage.Sequencer, txs ogmessage.Txis) error

type Delegate struct {
	TxCreator          *txmaker.OGTxCreator
	ReceivedNewTxsChan chan []ogmessage.Txi
	ReceivedNewTxChan  chan ogmessage.Txi
	TxPool             *core.TxPool
	Dag                *core.Dag
	OnNewTxiGenerated  []chan ogmessage.Txi
	InsertSyncBuffer   insertTxsFn
}

func (d *Delegate) GetTxNum() uint32 {
	return d.TxPool.GetTxNum()
}

func (d *Delegate) TooMoreTx() bool {
	if d.GetTxNum() > 6000 {
		return true
	}
	return false
}

func (c *Delegate) GenerateTx(r txmaker.SignedTxBuildRequest) (tx ogmessage.Txi, err error) {
	tx = c.TxCreator.NewSignedTx(r)

	if ok := c.TxCreator.SealTx(tx, nil); !ok {
		logrus.Warn("delegate failed to seal tx")
		err = fmt.Errorf("delegate failed to seal tx")
		return
	}
	logrus.WithField("tx", tx).Debugf("tx generated")
	return
}

func (c *Delegate) GenerateArchive(data []byte) (tx ogmessage.Txi, err error) {
	tx, err = c.TxCreator.NewArchiveWithSeal(data)
	if err != nil {
		logrus.WithField("tx", tx).Debugf("tx generated")
	}
	return
}

type SeqRequest struct {
	Issuer     common.Address
	PrivateKey crypto.PrivateKey
	Nonce      uint64
	Height     uint64
}

//discarded function
func (c *Delegate) GenerateSequencer(r SeqRequest) (seq *ogmessage.Sequencer, err error) {
	// TODO: why twice?
	seq, err, _ = c.TxCreator.GenerateSequencer(r.Issuer, r.Height, r.Nonce, &r.PrivateKey, nil)
	if err != nil {
		return
	}
	//if seq == nil && genAgain {
	//	seq, genAgain = c.TxCreator.GenerateSequencer(r.Issuer, r.Height, r.Nonce, &r.PrivateKey, nil)
	//}
	logrus.WithField("seq", seq).Infof("sequencer generated")
	if ok := c.TxCreator.SealTx(seq, &r.PrivateKey); !ok {
		err = fmt.Errorf("delegate failed to seal seq")
		return
	}
	logrus.WithField("seq", seq).Infof("sequencer connected")
	return seq, nil
}

func (c *Delegate) GetLatestAccountNonce(addr common.Address) (uint64, error) {
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

func (c *Delegate) GetLatestDagSequencer() *ogmessage.Sequencer {
	latestSeq := c.Dag.LatestSequencer()
	return latestSeq
}

func (c *Delegate) Announce(txi ogmessage.Txi) {
	for _, ch := range c.OnNewTxiGenerated {
		ch <- txi
	}
}

func (c *Delegate) JudgeNonce(me *account.Account) uint64 {

	var n uint64
	//NonceSelfDiscipline
	// fetch from db every time
	n, err := c.GetLatestAccountNonce(me.Address)
	me.SetNonce(n)
	if err != nil {
		// not exists, set to 0
		return 0
	} else {
		n, _ = me.ConsumeNonce()
		return n
	}
}
