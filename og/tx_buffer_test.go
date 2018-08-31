package og

import (
	"testing"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type dummyDag struct {
	dmap map[types.Hash]types.Txi
}

func (d *dummyDag) init() {
	tx := sampleTx("0x00", []string{})
	d.dmap[tx.GetBase().Hash] = tx
}

func (d *dummyDag) GetTx(hash types.Hash) types.Txi {
	return nil
}

type dummyTxPool struct {
	dmap map[types.Hash]types.Txi
}

func (d *dummyTxPool) Get(hash types.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func (d *dummyTxPool) AddRemoteTx(tx types.Txi) error {
	d.dmap[tx.GetBase().Hash] = tx
	return nil
}

type dummySyncer struct {
	dmap   map[types.Hash]types.Txi
	buffer *TxBuffer
}

func (d *dummySyncer) Know(tx types.Txi) {
	d.dmap[tx.GetBase().Hash] = tx
}

func (d *dummySyncer) Enqueue(hash types.Hash) {
	if v, ok := d.dmap[hash]; ok {
		d.buffer.AddTx(v)
	}
	logrus.Infof("Tx not found: %s", hash.Hex())
}

type dummyVerifier struct{}

func (d *dummyVerifier) VerifyHash(t types.Txi) bool {
	return true
}
func (d *dummyVerifier) VerifySignature(t types.Txi) bool {
	return true
}

func setup() *TxBuffer {
	buffer := TxBuffer{
		dag:      new(dummyDag),
		txPool:   new(dummyTxPool),
		syncer:   new(dummySyncer),
		verifier: new(dummyVerifier),
	}
	return &buffer
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

func TestBuffer(t *testing.T) {
	buffer := setup()
	m := buffer.syncer.(*dummySyncer)
	m.Know(sampleTx("0x01", []string{"0x00"}))
	m.Know(sampleTx("0x02", []string{"0x00"}))
	m.Know(sampleTx("0x03", []string{"0x00"}))
	m.Know(sampleTx("0x04", []string{"0x02"}))
	m.Know(sampleTx("0x05", []string{"0x01", "0x02"}))
	m.Know(sampleTx("0x06", []string{"0x02", "0x03"}))
	m.Know(sampleTx("0x07", []string{"0x04", "0x05"}))
	m.Know(sampleTx("0x08", []string{"0x05", "0x06"}))
	m.Know(sampleTx("0x09", []string{"0x07", "0x08"}))

	buffer.AddTx(sampleTx("0x0A", []string{"0x09"}))
}
