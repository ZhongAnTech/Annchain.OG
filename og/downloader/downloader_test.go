package downloader

import (
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/core"
	"math/big"
	"sync"
	"time"
	"github.com/annchain/OG/common/crypto"
	)

var (
	testKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testAddress = crypto.PubkeyToAddress(testKey.PublicKey)
)

// Reduce some of the parameters to make the tester faster.
func init() {
	MaxForkAncestry = uint64(10000)
	blockCacheItems = 1024
	fsHeaderContCheck = 500 * time.Millisecond
}


// downloadTester is a test simulator for mocking out local block chain.
type downloadTester struct {
	downloader *Downloader

	genesis *types.Sequencer   // Genesis blocks used by the tester and peers
	peerDb  ogdb.Database // Database of the peers containing all data

	ownHashes   []types.Hash                  // Hash chain belonging to the tester
	ownHeaders  map[types.Hash]*types.SequencerHeader  // Headers belonging to the tester
	ownBlocks   map[types.Hash]*types.Sequencer   // Blocks belonging to the tester
	ownChainTd  map[types.Hash]uint64     // id

	peerHashes   map[string][]types.Hash                  // Hash chain belonging to different test peers
	peerHeaders  map[string]map[types.Hash]*types.SequencerHeader  // Headers belonging to different test peers
	peerBlocks   map[string]map[types.Hash]*types.Sequencer // Blocks belonging to different test peers
	peerChainTds map[string]map[types.Hash]*big.Int       // Total difficulties of the blocks in the peer chains

	peerMissingStates map[string]map[types.Hash]bool // State entries that fast sync should not return

	lock sync.RWMutex
}


// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	testdb := ogdb.NewMemDatabase()
	genesis ,_ := core.DefaultGenesis()
	tester := &downloadTester{
		genesis:           genesis,
		peerDb:            testdb,
		ownHashes:         []types.Hash{genesis.GetTxHash()},
		ownHeaders:        map[types.Hash]*types.SequencerHeader{genesis.GetTxHash(): types.NewSequencerHead(genesis.GetTxHash(),genesis.Id)},
		ownBlocks:         map[types.Hash]*types.Sequencer{genesis.GetTxHash(): genesis},
		ownChainTd:        map[types.Hash]uint64{genesis.GetTxHash(): genesis.Number()},
		peerHashes:        make(map[string][]types.Hash),
		peerHeaders:       make(map[string]map[types.Hash]*types.SequencerHeader),
		peerBlocks:        make(map[string]map[types.Hash]*types.Sequencer),
		peerChainTds:      make(map[string]map[types.Hash]*big.Int),
		peerMissingStates: make(map[string]map[types.Hash]bool),
	}

	tester.downloader = New(FullSync, nil,nil,nil)

	return tester
}