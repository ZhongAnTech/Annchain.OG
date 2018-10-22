package og

import (
	"crypto/rand"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/og/downloader"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/discover"
	"github.com/annchain/OG/types"
	"testing"
)

var (
	testBankKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBank       = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testNetworkId  = uint64(101)
)

// newTestProtocolManager creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events.
func newTestHub(mode downloader.SyncMode) (*Hub, *ogdb.MemDatabase, error) {
	var (
		db               = ogdb.NewMemDatabase()
		genesis, balance = core.DefaultGenesis()
		config           = core.DagConfig{}
		dag              = core.NewDag(config, db)
	)
	if err := dag.Init(genesis, balance); err != nil {
		panic(err)
	}
	txConf := core.DefaultTxPoolConfig()
	txPool := core.NewTxPool(txConf, dag)
	txPool.Init(genesis)

	hubConf := DefaultHubConfig()
	hubConf.NetworkId = testNetworkId //for test
	hub := NewHub(&hubConf, mode, dag, txPool)
	/*
		syncConf := DefaultSyncerConfig()
		syncer := NewSyncer(&syncConf, hub)
		verfier := &GraphVerifier{
			Signer:       &crypto.SignerSecp256k1{},
			CryptoType:   crypto.CryptoTypeSecp256k1,
			Dag:          dag,
			TxPool:       txPool,
			MaxTxHash:    types.HexToHash("0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
			MaxMinedHash: types.HexToHash("0x00000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		}

		//bufConf := DefaultTxBufferConfig(syncer, txPool, dag, verfier)
		//txBuffer := NewTxBuffer(bufConf)
		//txBuffer.Hub = hub
		//hub.TxBuffer = txBuffer

		//dag.Start()
		//txPool.Start()

		//syncer.Start()
		txBuffer.Start()
	*/
	if hub == nil {
		return nil, nil, fmt.Errorf("hub init error")
	}
	hub.Start()

	return hub, db, nil
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
	*peer
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(name string, version int, h *Hub, shake bool) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id discover.NodeID
	rand.Read(id[:])

	peer := newPeer(version, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	go func() {
		select {
		case h.newPeerCh <- peer:
			errc <- h.handle(peer)
		case <-h.quitSync:
			errc <- p2p.DiscQuitting
		}
	}()
	tp := &testPeer{app: app, net: net, peer: peer}
	// Execute any implicitly requested handshakes and return
	if shake {
		var (
			genesis = h.Dag.Genesis()
			head    = h.Dag.LatestSequencer()
			id      = h.Dag.LatestSequencer().Number()
		)
		tp.handshake(nil, id, head.GetTxHash(), genesis.GetTxHash())
	}
	return tp, errc
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, seqId uint64, head types.Hash, genesis types.Hash) {
	msg := &StatusData{
		ProtocolVersion: uint32(p.version),
		NetworkId:       testNetworkId,
		CurrentId:       seqId,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
	}
	if err := p2p.ExpectMsg(p.app, uint64(StatusMsg), msg); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	data, _ := msg.MarshalMsg(nil)
	if err := p2p.Send(p.app, uint64(StatusMsg), data); err != nil {
		t.Fatalf("status send: %v", err)
	}
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}

func TestDatasize(t *testing.T) {
	var r types.Hash
	data, _ := r.MarshalMsg(nil)
	if len(data) == r.Msgsize() {
		t.Fatal("msg size not equal", "len data", len(data), "msgSize", r.Msgsize())
	}
}
