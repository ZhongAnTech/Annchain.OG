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
package downloader

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/protocol/dagmessage"
	"github.com/annchain/OG/og/types"
	core2 "github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	"math/big"
	"sync"
	"testing"
	"time"
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

	genesis *types.Sequencer // Genesis blocks used by the tester and peers
	peerDb  ogdb.Database    // Database of the peers containing all data

	ownHashes  types2.Hashes                               // Hash chain belonging to the tester
	ownHeaders map[types2.Hash]*dagmessage.SequencerHeader // Headers belonging to the tester
	ownBlocks  map[types2.Hash]*types.Sequencer            // Blocks belonging to the tester
	ownChainTd map[types2.Hash]uint64                      // id

	peerHashes   map[string]types2.Hashes                               // Hash chain belonging to different test peers
	peerHeaders  map[string]map[types2.Hash]*dagmessage.SequencerHeader // Headers belonging to different test peers
	peerBlocks   map[string]map[types2.Hash]*types.Sequencer            // Blocks belonging to different test peers
	peerChainTds map[string]map[types2.Hash]*big.Int                    // Total difficulties of the blocks in the peer chains

	peerMissingStates map[string]map[types2.Hash]bool // State entries that fast sync should not return

	lock sync.RWMutex
}

// newTester creates a new downloader test mocker.
func newTester() *downloadTester {
	testdb := ogdb.NewMemDatabase()
	genesis, _ := core2.DefaultGenesis(0)
	tester := &downloadTester{
		genesis:           genesis,
		peerDb:            testdb,
		ownHashes:         types2.Hashes{genesis.GetHash()},
		ownHeaders:        map[types2.Hash]*dagmessage.SequencerHeader{genesis.GetHash(): types.NewSequencerHead(genesis.GetHash(), genesis.Number())},
		ownBlocks:         map[types2.Hash]*types.Sequencer{genesis.GetHash(): genesis},
		ownChainTd:        map[types2.Hash]uint64{genesis.GetHash(): genesis.Number()},
		peerHashes:        make(map[string]types2.Hashes),
		peerHeaders:       make(map[string]map[types2.Hash]*dagmessage.SequencerHeader),
		peerBlocks:        make(map[string]map[types2.Hash]*types.Sequencer),
		peerChainTds:      make(map[string]map[types2.Hash]*big.Int),
		peerMissingStates: make(map[string]map[types2.Hash]bool),
	}

	tester.downloader = New(FullSync, nil, nil, nil)

	return tester
}

func TestHeaderEuqual(t *testing.T) {
	testHash, _ := types2.HexToHash("0xe6a07ee5c2fb20b07ec81f0b124b9b4428b8a96e99de01a440b5e0c4c25e22e3")
	head := types.NewSequencerHead(testHash, 1447)
	seq := &types.Sequencer{}
	seq.Height = 1447
	seq.Hash = testHash
	seq.Issuer = &common.Address{}
	seqHead := seq.GetHead()
	if head == seqHead {
		t.Fatal("head", head.StringFull(), " seqHead", seqHead.StringFull(), "struct  shoud not be  equal")
	}
	if !head.Equal(seqHead) {
		t.Fatal("head", head.StringFull(), " seqHead", seqHead.StringFull(), "content  shoud  be  equal")
	}
	t.Log("head", head, " seqHead", seqHead, "equal")
}
