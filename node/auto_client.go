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
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/goroutine"
	"math/rand"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

const (
	IntervalModeConstantInterval = "constant"
	IntervalModeRandom           = "random"
)

type AutoClient struct {
	SampleAccounts       []*account.SampleAccount
	MyIndex              int                    //only for debug
	MyAccount            *account.SampleAccount //if MyAccount is
	SequencerIntervalUs  int
	TxIntervalUs         int
	ArchiveInterValUs    int
	IntervalMode         string
	NonceSelfDiscipline  bool
	AutoSequencerEnabled bool
	CampainEnable        bool
	AutoTxEnabled        bool
	AutoArchiveEnabled   bool

	Delegate *Delegate

	ManualChan chan types.TxBaseType
	quit       chan bool

	pause    bool
	testMode bool

	wg sync.WaitGroup

	nonceLock   sync.RWMutex
	txLock      sync.RWMutex
	archiveLock sync.RWMutex
	NewRawTx    chan types.Txi
}

func (c *AutoClient) Init() {
	c.quit = make(chan bool)
	c.ManualChan = make(chan types.TxBaseType)
	c.NewRawTx = make(chan types.Txi)
}

func (c *AutoClient) SetTxIntervalUs(i int) {
	c.TxIntervalUs = i
}

func (c *AutoClient) nextArchiveSleepDuraiton() time.Duration {
	// tx duration selection
	var sleepDuration time.Duration
	switch c.IntervalMode {
	case IntervalModeConstantInterval:
		sleepDuration = time.Microsecond * time.Duration(c.ArchiveInterValUs)
	case IntervalModeRandom:
		sleepDuration = time.Microsecond * (time.Duration(rand.Intn(c.ArchiveInterValUs-1) + 1))
	default:
		panic(fmt.Sprintf("unkown IntervalMode : %s  ", c.IntervalMode))
	}
	return sleepDuration
}

func (c *AutoClient) nextSleepDuraiton() time.Duration {
	// tx duration selection
	var sleepDuration time.Duration
	switch c.IntervalMode {
	case IntervalModeConstantInterval:
		sleepDuration = time.Microsecond * time.Duration(c.TxIntervalUs)
	case IntervalModeRandom:
		sleepDuration = time.Microsecond * (time.Duration(rand.Intn(c.TxIntervalUs-1) + 1))
	default:
		panic(fmt.Sprintf("unkown IntervalMode : %s  ", c.IntervalMode))
	}
	return sleepDuration
}

func (c *AutoClient) fireManualTx(txType types.TxBaseType, force bool) {
	switch txType {
	case types.TxBaseTypeNormal:
		c.doSampleTx(force)
	case types.TxBaseTypeSequencer:
		c.doSampleSequencer(force)
	default:
		logrus.WithField("type", txType).Warn("Unknown TxBaseType")
	}
}

func (c *AutoClient) loop() {
	c.pause = true
	c.wg.Add(1)
	defer c.wg.Done()

	timerTx := time.NewTimer(c.nextSleepDuraiton())
	timerArchive := time.NewTimer(c.nextArchiveSleepDuraiton())
	tickerSeq := time.NewTicker(time.Microsecond * time.Duration(c.SequencerIntervalUs))
	logrus.Debug(c.SequencerIntervalUs, "  seq duration")
	if !c.AutoTxEnabled {
		timerTx.Stop()
	}
	if !c.AutoSequencerEnabled {
		tickerSeq.Stop()
	}

	if !c.AutoArchiveEnabled {
		timerArchive.Stop()
	}

	for {
		if c.pause {
			logrus.Trace("client paused")
			select {
			case <-time.After(time.Second):
				continue
			case <-c.quit:
				logrus.Debug("got quit signal")
				return
			case txType := <-c.ManualChan:
				c.fireManualTx(txType, true)
			}
		}
		logrus.Trace("client is working")
		select {
		case <-c.quit:
			c.pause = true
			logrus.Debug("got quit signal")
			return
		case txType := <-c.ManualChan:
			c.fireManualTx(txType, true)
		case <-timerTx.C:
			logrus.Debug("timer sample tx")
			if c.testMode {
				timerTx.Stop()
				continue
			}
		    if c.Delegate.TooMoreTx() {
				timerTx.Stop()
				continue
			}
			c.doSampleTx(false)
			timerTx.Reset(c.nextSleepDuraiton())
		case tx := <-c.NewRawTx:
			c.doRawTx(tx)
		case <-timerArchive.C:
			logrus.Debug("timer sample tx")
			if c.Delegate.TooMoreTx() {
				timerArchive.Stop()
				continue
			}
			c.doSampleArchive(false)
			timerArchive.Reset(c.nextArchiveSleepDuraiton())
		case <-tickerSeq.C:
			if c.testMode {
				timerTx.Stop()
				continue
			}
			logrus.Debug("timer sample seq")
			c.doSampleSequencer(false)
		}

	}
}

func (c *AutoClient) Start() {
	goroutine.New(c.loop)
}

func (c *AutoClient) Stop() {
	c.quit <- true
	c.Delegate.TxCreator.Stop()
	c.wg.Wait()
}

func (c *AutoClient) Pause() {
	c.pause = true
}

func (c *AutoClient) Resume() {
	c.pause = false
}

func (c *AutoClient) judgeNonce() uint64 {
	c.nonceLock.Lock()
	defer c.nonceLock.Unlock()

	var n uint64
	me := c.MyAccount
	if c.NonceSelfDiscipline {
		n, err := me.ConsumeNonce()
		if err == nil {
			return n
		}
	}

	// fetch from db every time
	n, err := c.Delegate.GetLatestAccountNonce(me.Address)
	me.SetNonce(n)
	if err != nil {
		// not exists, set to 0
		return 0
	} else {
		n, _ = me.ConsumeNonce()
		return n
	}
}

func (c *AutoClient) fireTxs(me types.Address) bool {
	m := viper.GetInt("auto_client.tx.interval_us")
	if m == 0 {
		m = 1000
	}
	logrus.WithField("micro", m).Info("sent interval ")
	for i := uint64(1); i < 1000000000; i++ {
		if c.pause {
			logrus.Info("tx generate stopped")
			return true
		}
		time.Sleep(time.Duration(m) * time.Microsecond)
		txi := c.Delegate.Dag.GetOldTx(me, i)
		if txi == nil {
			return true
		}
		c.Delegate.Announce(txi)
	}
	return true
}

var firstTx bool

func (c *AutoClient) doSampleTx(force bool) bool {
	if !force && !c.AutoTxEnabled {
		return false
	}

	me := c.MyAccount
	if !firstTx {
		txi := c.Delegate.Dag.GetOldTx(me.Address, 0)
		if txi != nil {
			logrus.WithField("txi", txi).Info("get start test tps")
			c.AutoTxEnabled = false
			c.AutoSequencerEnabled = false
			c.testMode = true
			firstTx = true
			c.Delegate.Announce(txi)
			return c.fireTxs(me.Address)
		}
		firstTx = true
	}
	c.txLock.RLock()
	defer c.txLock.RUnlock()

	tx, err := c.Delegate.GenerateTx(TxRequest{
		AddrFrom:   me.Address,
		AddrTo:     c.SampleAccounts[rand.Intn(len(c.SampleAccounts))].Address,
		Nonce:      c.judgeNonce(),
		Value:      math.NewBigInt(0),
		PrivateKey: me.PrivateKey,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to auto generate tx")
		return false
	}
	logrus.WithField("tx", tx).WithField("nonce", tx.GetNonce()).
		WithField("id", c.MyIndex).Trace("Generated tx")
	c.Delegate.Announce(tx)
	return true
}

type randomArchive struct {
	Name    string `json:"name"`
	RandInt uint64 `json:"rand_int"`
	Num     int    `json:"num"`
	From    []byte `json:"from"`
}

var archiveNum int

func (c *AutoClient) doSampleArchive(force bool) bool {
	if !force && !c.AutoArchiveEnabled {
		return false
	}
	c.archiveLock.RLock()
	defer c.archiveLock.RUnlock()
	r := randomArchive{
		Name:    fmt.Sprintf("%d", rand.Int31()),
		RandInt: rand.Uint64(),
		Num:     archiveNum,
		From:    c.MyAccount.Address.ToBytes()[:5],
	}
	archiveNum++
	data, _ := json.Marshal(&r)
	tx, err := c.Delegate.GenerateArchive(data)
	if err != nil {
		logrus.WithError(err).Error("failed to auto generate tx")
		return false
	}
	logrus.WithField("tx", tx).WithField("nonce", tx.GetNonce()).
		WithField("id", c.MyIndex).Trace("Generated tx")
	c.Delegate.Announce(tx)
	return true
}

func (c *AutoClient) doRawTx(txi types.Txi) bool {
	if !c.CampainEnable {
		return false
	}
	me := c.MyAccount
	txi.GetBase().PublicKey = me.PublicKey.Bytes
	txi.GetBase().AccountNonce = c.judgeNonce()
	if txi.GetType() == types.TxBaseTypeCampaign {
		cp := txi.(*types.Campaign)
		cp.Issuer = me.Address
	} else if txi.GetType() == types.TxBaseTypeTermChange {
		cp := txi.(*types.TermChange)
		cp.Issuer = me.Address
	}
	s := crypto.NewSigner(me.PublicKey.Type)
	txi.GetBase().Signature = s.Sign(me.PrivateKey, txi.SignatureTargets()).Bytes
	if ok := c.Delegate.TxCreator.SealTx(txi, &me.PrivateKey); !ok {
		logrus.Warn("delegate failed to seal tx")
		return false
	}

	logrus.WithField("tx", txi).WithField("nonce", txi.GetNonce()).
		WithField("id", c.MyIndex).Trace("Generated txi")
	c.Delegate.Announce(txi)
	return true
}

func (c *AutoClient) doSampleSequencer(force bool) bool {
	if !force && !c.AutoSequencerEnabled {
		return false
	}
	me := c.MyAccount
	if !firstTx {
		txi := c.Delegate.Dag.GetOldTx(me.Address, 0)
		if txi != nil {
			c.AutoSequencerEnabled = false
			return true
		}
	}

	seq, err := c.Delegate.GenerateSequencer(SeqRequest{
		Issuer:     me.Address,
		Height:     c.Delegate.GetLatestDagSequencer().Height + 1,
		Nonce:      c.judgeNonce(),
		PrivateKey: me.PrivateKey,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to auto generate seq")
		return false
	}
	logrus.WithField("seq", seq).WithField("nonce", seq.GetNonce()).
		WithField("id", c.MyIndex).WithField("dump ", seq.Dump()).Debug("Generated seq")
	c.Delegate.Announce(seq)
	return true
}
