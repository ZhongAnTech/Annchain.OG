package ogcore_test

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/common/utilfuncs"
	"github.com/annchain/OG/eventbus"
	"github.com/annchain/OG/ffchan"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/communication"
	"github.com/annchain/OG/ogcore/events"
	"github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/message"
	"github.com/sirupsen/logrus"

	"github.com/annchain/gcache"
)

type dummyDag struct {
	dmap map[common.Hash]types.Txi
}

func (d *dummyDag) IsTxExists(hash common.Hash) bool {
	_, ok := d.dmap[hash]
	return ok
}

func (d *dummyDag) IsAddressExists(addr common.Address) bool {
	panic("implement me")
}

func (d *dummyDag) Push(batch *ledger.ConfirmBatch) error {
	d.dmap[batch.Seq.Hash] = batch.Seq
	for _, tx := range batch.Txs {
		d.dmap[tx.GetHash()] = tx
	}
	return nil
}

func (d *dummyDag) Name() string {
	return "dummyDag"
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
	return d.dmap[common.HexToHash("0x00")].(*types.Sequencer)
}

func (d *dummyDag) GetSequencer(hash common.Hash, id uint64) *types.Sequencer {
	return nil
}

func (d *dummyDag) Genesis() *types.Sequencer {
	return d.dmap[common.HexToHash("0x00")].(*types.Sequencer)
}

func (d *dummyDag) InitDefault() {
	d.dmap = make(map[common.Hash]types.Txi)
	tx := sampleSequencer("0x00", []string{}, 0)
	tx.Weight = 1
	d.dmap[tx.GetHash()] = tx
}

func (d *dummyDag) GetTx(hash common.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func sampleSequencer(selfHash string, parentsHash []string, nonce uint64) *types.Sequencer {
	parents, err := common.HexStringsToHashes(parentsHash)
	utilfuncs.PanicIfError(err, "sampleseq bad hex")

	seq := &types.Sequencer{
		Hash:         common.HexToHash(selfHash),
		ParentsHash:  parents,
		Height:       0,
		MineNonce:    0,
		AccountNonce: nonce,
		Issuer:       common.Address{},
		Signature:    nil,
		PublicKey:    nil,
		StateRoot:    common.Hash{},
		Weight:       0,
	}
	return seq
}

func sampleTx(selfHash string, parentsHash []string, nonce uint64) *types.Tx {
	parents, err := common.HexStringsToHashes(parentsHash)
	utilfuncs.PanicIfError(err, "sampletx bad hex")
	tx := &types.Tx{
		Hash:         common.HexToHash(selfHash),
		ParentsHash:  parents,
		AccountNonce: nonce,
		Value:        math.NewBigInt(0),
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, common.HexToHash(h))
	}
	return tx
}

type dummySyncer struct {
	EventBus            eventbus.EventBus
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
	v, ok := d.dmap[evt.Hash]
	if ok {
		// we already have this tx.
		logrus.WithField("tx", v).Debug("syncer found new tx")
		go d.EventBus.Route(&events.TxReceivedEvent{
			Tx: v,
		})
	}
}
func (o *dummySyncer) HandlerDescription(ev eventbus.EventType) string {
	return "N/A"
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
	d.dmap[tx.GetHash()] = tx
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
	tx := sampleTx("0x01", []string{"0x00"}, 0)
	d.dmap[tx.GetHash()] = tx
}

func (d *dummyTxPool) Get(hash common.Hash) types.Txi {
	if v, ok := d.dmap[hash]; ok {
		return v
	}
	return nil
}

func (d *dummyTxPool) AddRemoteTx(tx types.Txi, b bool) error {
	d.dmap[tx.GetHash()] = tx
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

type DummyOgPeerCommunicator struct {
	Myid        int
	PeerPipeIns []chan *communication.OgMessageEvent
	pipeIn      chan *communication.OgMessageEvent
	pipeOut     chan *communication.OgMessageEvent
}

func (o DummyOgPeerCommunicator) Broadcast(msg message.OgMessage) {
	for i, peerChan := range o.PeerPipeIns {
		logrus.WithField("peer", i).WithField("me", o.Myid).WithField("type", msg.GetType()).Debug("broadcasting message")
		me := &communication.OgPeer{
			Id:             o.Myid,
			PublicKey:      crypto.PublicKey{},
			Address:        common.Address{},
			PublicKeyBytes: nil,
		}
		go func(i int, peerChan chan *communication.OgMessageEvent) {
			//<- ffchan.NewTimeoutSenderShort(o.PeerPipeIns[peer.Id], msg, "dkg").C
			ffchan.NewTimeoutSenderShort(peerChan, &communication.OgMessageEvent{
				Message: msg,
				Peer:    me,
			}, "peercomm")
			//peerChan <- &communication.OgMessageEvent{
			//	Message: msg,
			//	Peer:    me,
			//}
		}(i, peerChan)
	}
}

func (o DummyOgPeerCommunicator) GetPipeIn() chan *communication.OgMessageEvent {
	return o.pipeIn
}

func (o DummyOgPeerCommunicator) GetPipeOut() chan *communication.OgMessageEvent {
	return o.pipeOut
}

func NewDummyOgPeerCommunicator(myid int, incoming chan *communication.OgMessageEvent, peers []chan *communication.OgMessageEvent) *DummyOgPeerCommunicator {
	d := &DummyOgPeerCommunicator{
		PeerPipeIns: peers,
		Myid:        myid,
		pipeIn:      incoming,
		pipeOut:     make(chan *communication.OgMessageEvent, 100), // must be big enough to avoid blocking issue
	}
	return d
}

func (o DummyOgPeerCommunicator) Multicast(msg message.OgMessage, peers []*communication.OgPeer) {
	for _, peer := range peers {
		logrus.WithField("peer", peer.Id).WithField("me", o.Myid).Debug("multicasting message")
		go func(peer *communication.OgPeer) {
			//<- ffchan.NewTimeoutSenderShort(o.PeerPipeIns[peer.Id], msg, "dkg").C
			o.PeerPipeIns[peer.Id] <- &communication.OgMessageEvent{
				Message: msg,
				Peer:    peer,
			}
		}(peer)
	}
}

func (o DummyOgPeerCommunicator) Unicast(msg message.OgMessage, peer *communication.OgPeer) {
	logrus.Debug("unicasting by DummyOgPeerCommunicator")
	go func() {
		//ffchan.NewTimeoutSenderShort(d.PeerPipeIns[peer.Id], msg, "bft")
		o.PeerPipeIns[peer.Id] <- &communication.OgMessageEvent{
			Message: msg,
			Peer:    &communication.OgPeer{Id: o.Myid},
		}
	}()
}

func (d *DummyOgPeerCommunicator) Run() {
	logrus.Info("DummyOgPeerCommunicator running")
	go func() {
		for {
			v := <-d.pipeIn
			//vv := v.Message.(bft.BftMessage)
			logrus.WithField("type", v.Message.GetType()).Debug("DummyOgPeerCommunicator received a message")
			d.pipeOut <- v
		}
	}()
}
