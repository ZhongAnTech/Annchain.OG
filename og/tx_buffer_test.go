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
package og

import (
	"fmt"
	"testing"
	"time"

	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/annchain/gcache"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
)

type dummyDag struct {
	dmap map[common.Hash]types.Txi
}

func (d *dummyDag) GetHeight() uint64 {
	return 0
}

func (d *dummyDag) GetSequencerByHeight(id uint64) *tx_types.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencerByHash(hash common.Hash) *tx_types.Sequencer {
	return nil
}

func (d *dummyDag) GetBalance(address common.Address, tokenId int32) *math.BigInt {
	return math.NewBigInt(0)
}

func (d *dummyDag) GetTxByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (d *dummyDag) GetTxisByNumber(id uint64) types.Txis {
	return nil
}

func (d *dummyDag) GetTestTxisByNumber(id uint64) types.Txis {
	return nil
}

func (d *dummyDag) LatestSequencer() *tx_types.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencer(hash common.Hash, id uint64) *tx_types.Sequencer {
	return nil
}

func (d *dummyDag) Genesis() *tx_types.Sequencer {
	return nil
}

func (d *dummyDag) init() {
	d.dmap = make(map[common.Hash]types.Txi)
	tx := sampleTx("0x00", []string{})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyDag) GetTx(hash common.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

type dummyTxPool struct {
	dmap map[common.Hash]types.Txi
}

func (d *dummyTxPool) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (d *dummyTxPool) RegisterOnNewTxReceived(c chan types.Txi, s string, b bool) {
	return
}

func (d *dummyTxPool) GetMaxWeight() uint64 {
	return 0
}

func (d *dummyTxPool) init() {
	d.dmap = make(map[common.Hash]types.Txi)
	tx := sampleTx("0x01", []string{"0x00"})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyTxPool) Get(hash common.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func (d *dummyTxPool) AddRemoteTx(tx types.Txi, b bool) error {
	d.dmap[tx.GetTxHash()] = tx
	return nil
}

func (d *dummyTxPool) IsLocalHash(hash common.Hash) bool {
	return false
}

type dummySyncer struct {
	dmap                map[common.Hash]types.Txi
	buffer              *TxBuffer
	acquireTxDedupCache gcache.Cache
}

func (d *dummySyncer) ClearQueue() {
	for k := range d.dmap {
		delete(d.dmap, k)
	}
}

func (d *dummySyncer) Know(tx types.Txi) {
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummySyncer) IsCachedHash(hash common.Hash) bool {
	return false
}

func (d *dummySyncer) Enqueue(hash *common.Hash, childHash common.Hash, b bool) {
	if _, err := d.acquireTxDedupCache.Get(*hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	d.acquireTxDedupCache.Set(hash, struct{}{})

	if v, ok := d.dmap[*hash]; ok {
		<-ffchan.NewTimeoutSenderShort(d.buffer.ReceivedNewTxChan, v, "test").C
		logrus.WithField("hash", hash).Infof("syncer added tx")
		logrus.WithField("hash", hash).Infof("syncer returned tx")
	} else {
		logrus.WithField("hash", hash).Infof("syncer does not know tx")
	}

}

type dummyVerifier struct{}

func (d *dummyVerifier) Verify(t types.Txi) bool {
	return true
}

func (d *dummyVerifier) Name() string {
	return "dumnmy verifier"
}

func (d *dummyVerifier) String() string {
	return d.Name()
}

func (d *dummyVerifier) Independent() bool {
	return false
}

func setup() *TxBuffer {
	ver := new(dummyVerifier)
	buffer := NewTxBuffer(TxBufferConfig{
		Verifiers:                        []Verifier{ver},
		DependencyCacheMaxSize:           20,
		TxPool:                           new(dummyTxPool),
		Dag:                              new(dummyDag),
		Syncer:                           new(dummySyncer),
		DependencyCacheExpirationSeconds: 60,
		NewTxQueueSize:                   100,
		KnownCacheMaxSize:                10000,
		KnownCacheExpirationSeconds:      30,
	})
	buffer.Syncer.(*dummySyncer).dmap = make(map[common.Hash]types.Txi)
	buffer.Syncer.(*dummySyncer).buffer = buffer
	buffer.Syncer.(*dummySyncer).acquireTxDedupCache = gcache.New(100).Simple().
		Expiration(time.Second * 10).Build()
	buffer.dag.(*dummyDag).init()
	buffer.txPool.(*dummyTxPool).init()
	return buffer
}

func sampleTx(selfHash string, parentsHash []string) *tx_types.Tx {
	tx := &tx_types.Tx{TxBase: types.TxBase{
		ParentsHash: common.Hashes{},
		Type:        types.TxBaseTypeNormal,
		Hash:        common.HexToHash(selfHash),
	},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, common.HexToHash(h))
	}
	return tx
}

func doTest(buffer *TxBuffer) {
	buffer.Start()

	if buffer.dependencyCache.Len(true) != 0 {
		for k, v := range buffer.dependencyCache.GetALL(true) {
			for k1 := range v.(map[common.Hash]types.Txi) {
				logrus.Warnf("not fulfilled: %s <- %s", k.(common.Hash), k1)
			}
		}
	}
}

func TestBuffer(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	assert.Equal(t, buffer.dependencyCache.Len(true), 0)
}

func TestBufferMissing3(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	//m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	tx := sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	// missing 5,7,8,9
	assert.Equal(t, buffer.dependencyCache.Len(true), 5)
}

func TestBufferCache(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.Syncer.(*dummySyncer)
	tx := sampleTx("0x0A", []string{"0x09"})
	<-ffchan.NewTimeoutSenderShort(buffer.ReceivedNewTxChan, tx, "test").C
	buffer.Start()
	success := false
	for i := 0; i < 8; i++ {
		time.Sleep(time.Second * 2)
		// query request cache
		_, err := m.acquireTxDedupCache.Get(common.HexToHash("0x09"))
		if err != nil {
			// not found
			logrus.Debug("not in cache")
			success = true
			break
		} else {
			logrus.Debug("in cache")
		}
	}
	buffer.Stop()
	assert.Equal(t, success, true)
}

func TestLocalHash(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{})
	buffer := setup()
	tx2 := sampleTx("0x02", []string{"0x00"})
	tx3 := sampleTx("0x03", []string{"0x00"})
	buffer.txPool.AddRemoteTx(tx2, true)
	if !buffer.isLocalHash(tx2.GetTxHash()) {
		t.Fatal("is localhash")
	}
	if buffer.isLocalHash(tx3.GetTxHash()) {
		t.Fatal("is not localhash")
	}
}

func TestTxBuffer_Handle(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.TraceLevel)
	ver := &TxFormatVerifier{
		NoVerifyMaxTxHash: true,
		NoVerifyMindHash:  true,
	}
	buffer := NewTxBuffer(TxBufferConfig{
		Verifiers:                        []Verifier{ver},
		DependencyCacheMaxSize:           20,
		TxPool:                           new(dummyTxPool),
		DependencyCacheExpirationSeconds: 60,
		NewTxQueueSize:                   100,
		KnownCacheMaxSize:                10000,
		KnownCacheExpirationSeconds:      30,
	})
	pub, priv := crypto.Signer.RandomKeyPair()
	from := pub.Address()
	N := 20
	var txs types.Txis
	for i := 0; i < N; i++ {
		tx := tx_types.RandomTx()
		tx.Height = 1
		tx.Weight = tx.Weight % uint64(N)
		tx.SetSender(from)
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes
		tx.PublicKey = pub.Bytes
		tx.Hash = tx.CalcTxHash()
		txs = append(txs, tx)
	}
	logrus.Debug("handle txis start", txs)
	buffer.handleTxs(txs)
	for i := 0; i < N; i++ {

	}
}
