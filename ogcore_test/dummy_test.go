package ogcore_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore"
	"github.com/annchain/OG/ogcore/events"
	"github.com/sirupsen/logrus"

	"github.com/annchain/gcache"
)

type dummyDag struct {
	dmap map[common.Hash]types.Txi
}

func (d *dummyDag) Name() string {
	panic("implement me")
}

func (d *dummyDag) GetHeightTxs(height uint64, offset uint32, limit uint32) []types.Txi {
	var txs []types.Txi
	for _, v := range d.dmap {
		txs = append(txs, v)
	}
	if uint32(len(txs)) > offset+limit {
		return txs[offset : offset+limit]
	} else {
		return txs[offset:]
	}
}

func (d *dummyDag) IsLocalHash(hash common.Hash) bool {
	_, ok := d.dmap[hash]
	return ok
}

func (d *dummyDag) GetHeight() uint64 {
	return 0
}

func (d *dummyDag) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, nil
}

func (d *dummyDag) GetSequencerByHeight(id uint64) *types.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencerByHash(hash common.Hash) *types.Sequencer {
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

func (d *dummyDag) LatestSequencer() *types.Sequencer {
	return nil
}

func (d *dummyDag) GetSequencer(hash common.Hash, id uint64) *types.Sequencer {
	return nil
}

func (d *dummyDag) Genesis() *types.Sequencer {
	return nil
}

func (d *dummyDag) InitDefault() {
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

func sampleTx(selfHash string, parentsHash []string) *types.Tx {
	tx := &types.Tx{
		Hash:        common.HexToHash(selfHash),
		ParentsHash: common.Hashes{},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, common.HexToHash(h))
	}
	return tx
}

type dummySyncer struct {
	EventBus            ogcore.EventBus
	dmap                map[common.Hash]*types.Tx
	acquireTxDedupCache gcache.Cache
}

func (t *dummySyncer) Name() string {
	return "dummySyncer"
}

func (d *dummySyncer) InitDefault() {
	d.dmap = make(map[common.Hash]*types.Tx)
}

func (d *dummySyncer) HandleEvent(ev eventbus.Event) {
	evt := ev.(*events.NeedSyncEvent)
	v, ok := d.dmap[evt.ParentHash]
	if ok {
		// we already have this tx.
		logrus.WithField("tx", v).Debug("syncer found new tx")
		go d.EventBus.Route(&events.TxReceivedEvent{
			Tx: v,
		})
	}
}

func (d *dummySyncer) ClearQueue() {
	for k := range d.dmap {
		delete(d.dmap, k)
	}
}

func (d *dummySyncer) SyncHashList(seqHash common.Hash) {
	return
}

// Know will let dummySyncer pretend it knows some tx
func (d *dummySyncer) Know(tx *types.Tx) {
	d.dmap[tx.GetTxHash()] = tx
}

func (d *dummySyncer) IsCachedHash(hash common.Hash) bool {
	return false
}

func (d *dummySyncer) Enqueue(hash *common.Hash, childHash common.Hash, b bool) {
	//if _, err := d.acquireTxDedupCache.Get(*hash); err == nil {
	//	logrus.WithField("Hash", hash).Debugf("duplicate sync task")
	//	return
	//}
	//d.acquireTxDedupCache.Set(hash, struct{}{})
	//
	//if v, ok := d.dmap[*hash]; ok {
	//	<-ffchan.NewTimeoutSenderShort(d.buffer.ReceivedNewTxChan, v, "test").C
	//	logrus.WithField("Hash", hash).Infof("syncer added tx")
	//	logrus.WithField("Hash", hash).Infof("syncer returned tx")
	//} else {
	//	logrus.WithField("Hash", hash).Infof("syncer does not know tx")
	//}

}

type dummyTxPool struct {
	dmap map[common.Hash]types.Txi
}

func (d *dummyTxPool) GetLatestNonce(addr common.Address) (uint64, error) {
	return 0, fmt.Errorf("not supported")
}

func (p *dummyTxPool) IsBadSeq(seq *types.Sequencer) error {
	return nil
}

func (d *dummyTxPool) RegisterOnNewTxReceived(c chan types.Txi, s string, b bool) {
	return
}

func (d *dummyTxPool) GetMaxWeight() uint64 {
	return 0
}

func (d *dummyTxPool) GetByNonce(addr common.Address, nonce uint64) types.Txi {
	return nil
}

func (d *dummyTxPool) InitDefault() {
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
