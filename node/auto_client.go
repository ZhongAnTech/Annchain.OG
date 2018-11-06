package node

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

const (
	IntervalModeConstantInterval = "constant"
	IntervalModeRandom           = "random"
)

type AutoClient struct {
	SampleAccounts []*account.SampleAccount
	MyAccountIndex int

	SequencerIntervalMs  int
	TxIntervalMs         int
	IntervalMode         string
	NonceSelfDiscipline  bool
	AutoSequencerEnabled bool
	AutoTxEnabled        bool

	Delegate *Delegate

	ManualChan chan types.TxBaseType
	quit       chan bool

	pause bool

	wg sync.WaitGroup

	nonceLock sync.RWMutex
}

func (c *AutoClient) Init() {
	c.quit = make(chan bool)
	c.ManualChan = make(chan types.TxBaseType)
}

func (c *AutoClient) nextSleepDuraiton() time.Duration {
	// tx duration selection
	var sleepDuration time.Duration
	switch c.IntervalMode {
	case IntervalModeConstantInterval:
		sleepDuration = time.Millisecond * time.Duration(c.TxIntervalMs)
	case IntervalModeRandom:
		sleepDuration = time.Millisecond * (time.Duration(rand.Intn(c.TxIntervalMs-1) + 1))
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
	tickerSeq := time.NewTicker(time.Millisecond * time.Duration(c.SequencerIntervalMs))

	if !c.AutoTxEnabled {
		timerTx.Stop()
	}
	if !c.AutoSequencerEnabled {
		tickerSeq.Stop()
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
			logrus.Debug("got quit signal")
			return
		case txType := <-c.ManualChan:
			c.fireManualTx(txType, true)
		case <-timerTx.C:
			logrus.Debug("timer sample tx")
			c.doSampleTx(false)
			timerTx.Reset(c.nextSleepDuraiton())
		case <-tickerSeq.C:
			logrus.Debug("timer sample seq")
			c.doSampleSequencer(false)
		}

	}
}

func (c *AutoClient) Start() {
	go c.loop()
}

func (c *AutoClient) Stop() {
	c.quit <- true
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
	me := c.SampleAccounts[c.MyAccountIndex]
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

func (c *AutoClient) doSampleTx(force bool) bool {
	if !force && !c.AutoTxEnabled {
		return false
	}

	me := c.SampleAccounts[c.MyAccountIndex]

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
		WithField("id", c.MyAccountIndex).Info("Generated tx")
	c.Delegate.Announce(tx)
	return true
}
func (c *AutoClient) doSampleSequencer(force bool) bool {
	if !force && !c.AutoSequencerEnabled {
		return false
	}
	me := c.SampleAccounts[c.MyAccountIndex]

	seq, err := c.Delegate.GenerateSequencer(SeqRequest{
		Issuer:     me.Address,
		SequenceID: c.Delegate.GetLatestDagSequencer().Id + 1,
		Nonce:      c.judgeNonce(),
		Hashes:     []types.Hash{},
		PrivateKey: me.PrivateKey,
	})
	if err != nil {
		logrus.WithError(err).Error("failed to auto generate seq")
		return false
	}
	c.Delegate.Announce(seq)
	return true
}
