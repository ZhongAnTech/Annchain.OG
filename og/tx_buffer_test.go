package og

import (
	"github.com/annchain/OG/types"
	"github.com/bluele/gcache"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

type dummyDag struct {
	dmap map[types.Hash]types.Txi
}

func (d *dummyDag) init() {
	d.dmap = make(map[types.Hash]types.Txi)
	tx := sampleTx("0x00", []string{})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyDag) GetTx(hash types.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

type dummyTxPool struct {
	dmap map[types.Hash]types.Txi
}

func (d *dummyTxPool) init() {
	d.dmap = make(map[types.Hash]types.Txi)
	tx := sampleTx("0x01", []string{"0x00"})
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummyTxPool) Get(hash types.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func (d *dummyTxPool) AddRemoteTx(tx types.Txi) error {
	d.dmap[tx.GetTxHash()] = tx
	return nil
}

type dummySyncer struct {
	dmap                map[types.Hash]types.Txi
	buffer              *TxBuffer
	acquireTxDedupCache gcache.Cache
}

func (d *dummySyncer) Know(tx types.Txi) {
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummySyncer) Enqueue(hash types.Hash) {
	if _, err := d.acquireTxDedupCache.Get(hash); err == nil {
		logrus.WithField("hash", hash).Debugf("duplicate sync task")
		return
	}
	d.acquireTxDedupCache.Set(hash, struct{}{})

	if v, ok := d.dmap[hash]; ok {
		d.buffer.AddTx(v)
		logrus.WithField("hash", hash).Infof("syncer added tx")
		logrus.WithField("hash", hash).Infof("syncer returned tx")
	} else {
		logrus.WithField("hash", hash).Infof("syncer does not know tx")
	}

}

type dummyVerifier struct{}

func (d *dummyVerifier) VerifyHash(t types.Txi) bool {
	return true
}
func (d *dummyVerifier) VerifySignature(t types.Txi) bool {
	return true
}
func (d *dummyVerifier) VerifySourceAddress(t types.Txi) bool {
	return true
}

func setup() *TxBuffer {
	buffer := NewTxBuffer(TxBufferConfig{
		Verifier:               new(dummyVerifier),
		DependencyCacheMaxSize: 20,
		TxPool:                 new(dummyTxPool),
		Dag:                    new(dummyDag),
		Syncer:                 new(dummySyncer),
		DependencyCacheExpirationSeconds: 60,
		NewTxQueueSize:                   100,
	})

	buffer.syncer.(*dummySyncer).dmap = make(map[types.Hash]types.Txi)
	buffer.syncer.(*dummySyncer).buffer = buffer
	buffer.syncer.(*dummySyncer).acquireTxDedupCache = gcache.New(100).Simple().
		Expiration(time.Second * 10).Build()
	buffer.dag.(*dummyDag).init()
	buffer.txPool.(*dummyTxPool).init()
	return buffer
}

func sampleTx(selfHash string, parentsHash []string) *types.Tx {
	tx := &types.Tx{TxBase: types.TxBase{
		ParentsHash: []types.Hash{},
		Type:        types.TxBaseTypeNormal,
		Hash:        types.HexToHash(selfHash),
	},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, types.HexToHash(h))
	}
	return tx
}

func doTest(buffer *TxBuffer) {
	buffer.Start()

	if buffer.dependencyCache.Len() != 0 {
		for k, v := range buffer.dependencyCache.GetALL() {
			for k1 := range v.(map[types.Hash]types.Txi) {
				logrus.Warnf("not fulfilled: %s <- %s", k.(types.Hash), k1)
			}
		}
	}
}

func TestBuffer(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	buffer.AddTx(sampleTx("0x0A", []string{"0x09"}))
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	assert.Equal(t, buffer.dependencyCache.Len(), 0)
}

func TestBufferMissing3(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.syncer.(*dummySyncer)
	//m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	//m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02", "0x03"}))
	m.Know(sampleTx("0x06", []string{"0x02"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))
	buffer.AddTx(sampleTx("0x0A", []string{"0x09"}))
	//buffer.AddTx(sampleTx("0x09", []string{"0x04"}))

	doTest(buffer)
	time.Sleep(time.Second * 3)
	buffer.Stop()
	// missing 5,7,8,9
	assert.Equal(t, buffer.dependencyCache.Len(), 5)
}

func TestBufferCache(t *testing.T) {
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)
	buffer := setup()
	m := buffer.syncer.(*dummySyncer)
	buffer.AddTx(sampleTx("0x0A", []string{"0x09"}))
	buffer.Start()
	success := false
	for i := 0; i < 8; i++ {
		time.Sleep(time.Second * 2)
		// query request cache
		_, err := m.acquireTxDedupCache.Get(types.HexToHash("0x09"))
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
