package og

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/common/files"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/math"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/minio/sha256-simd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"math/rand"
	"path"
	"strconv"
	"strings"
)

type BlockContentType int

const (
	BlockContentTypeInt BlockContentType = iota
)

type BlockContent interface {
	GetType() BlockContentType
	String() string
	GetHash() og_interface.Hash
}

type Ledger interface {
	CurrentHeight() int64
	CurrentCommittee() *consensus_interface.Committee
	GetBlock(height int64) BlockContent
	AddBlock(height int64, block BlockContent)
}

type IntArrayBlockContent struct {
	Values []int
}

func (i IntArrayBlockContent) GetType() BlockContentType {
	return BlockContentTypeInt
}

func (i IntArrayBlockContent) String() string {
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(i.Values)), "_"), "[]")
}

func (i IntArrayBlockContent) GetHash() og_interface.Hash {
	s, err := json.Marshal(i.Values)
	utilfuncs.PanicIfError(err, "get block hash")
	sum := sha256.Sum256(s)
	h := &og_interface.Hash32{}
	h.FromBytes(sum[:])
	return h
}

type IntArrayLedger struct {
	height        int64
	genesis       *Genesis
	blockContents map[int64]BlockContent // height-content
}

func (d *IntArrayLedger) Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string) {
	panic("implement me")
}

func (d *IntArrayLedger) GetState(blockId string) (stateId string) {
	panic("implement me")
}

func (d *IntArrayLedger) Commit(blockId string) {
	panic("implement me")
}

func (d *IntArrayLedger) GetHighQC() *consensus_interface.QC {
	panic("implement me")
}

func (d *IntArrayLedger) SaveConsensusState(state *consensus_interface.ConsensusState) {
	panic("implement me")
}

func (d *IntArrayLedger) LoadConsensusState() *consensus_interface.ConsensusState {
	panic("implement me")
}

func (d *IntArrayLedger) AddBlock(height int64, block BlockContent) {
	d.blockContents[height] = block
	d.height = math.BiggerInt64(height, d.height)
}

func (d *IntArrayLedger) GetBlock(height int64) BlockContent {
	return d.blockContents[height]
}

func (d *IntArrayLedger) AddRandomBlock(height int64) {
	size := 1 + rand.Intn(50)
	vs := make([]int, size)
	for i := 0; i < size; i++ {
		vs[i] = rand.Intn(100000)
	}
	d.AddBlock(height, &IntArrayBlockContent{
		Values: vs,
	})
}

func (d *IntArrayLedger) InitDefault() {
	d.blockContents = make(map[int64]BlockContent)
}

// StaticSetup supposely will load ledger from disk.
func (d *IntArrayLedger) StaticSetup() {
	d.height = 0
	rootHash := &og_interface.Hash32{}
	rootHash.FromHexNoError("0x00")

	d.LoadGenesis()
	//for i := 1; i < 10000; i ++{
	//	d.AddRandomBlock(int64(i))
	//}

	//d.AddRandomBlock(2)
	//d.AddRandomBlock(3)
	//d.AddRandomBlock(4)
	//d.AddRandomBlock(5)
	//d.DumpLedger()
	d.loadLedger()
}

func (d *IntArrayLedger) LoadGenesis() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "config")
	byteContent, err := ioutil.ReadFile(path.Join(datadir, "genesis.json"))
	if err != nil {
		return
	}

	gs := &GenesisStore{}
	err = json.Unmarshal(byteContent, gs)
	if err != nil {
		return
	}

	// convert to genesis
	hash := &og_interface.Hash32{}
	err = hash.FromHex(gs.RootSequencerHash)
	utilfuncs.PanicIfError(err, "root seq hash")

	peers := []*consensus_interface.CommitteeMember{}
	unmarshaller := crypto.PubKeyUnmarshallers[pb.KeyType_Secp256k1]

	for _, v := range gs.FirstCommittee.Peers {
		pubKeyBytes, err := hexutil.FromHex(v.PublicKey)
		utilfuncs.PanicIfError(err, "pubkey")

		pubKey, err := unmarshaller(pubKeyBytes)
		utilfuncs.PanicIfError(err, "pubkey")

		peers = append(peers, &consensus_interface.CommitteeMember{
			PeerIndex: v.PeerIndex,
			MemberId:  v.MemberId,
			PublicKey: pubKey,
		})
	}

	g := &Genesis{
		RootSequencerHash: hash,
		FirstCommittee: &consensus_interface.Committee{
			Peers:   peers,
			Version: gs.FirstCommittee.Version,
		},
	}
	d.genesis = g
}

func (d *IntArrayLedger) DumpGenesis() {

	peers := []CommitteeMemberStore{}
	for _, v := range d.genesis.FirstCommittee.Peers {
		pubKeyBytes, err := v.PublicKey.Raw()
		utilfuncs.PanicIfError(err, "pubkey raw")
		peers = append(peers, CommitteeMemberStore{
			PeerIndex: v.PeerIndex,
			MemberId:  v.MemberId,
			PublicKey: hexutil.ToHex(pubKeyBytes),
		})
	}

	gs := &GenesisStore{
		RootSequencerHash: d.genesis.RootSequencerHash.HashString(),
		FirstCommittee: CommitteeStore{
			Version: d.genesis.FirstCommittee.Version,
			Peers:   peers,
		},
	}

	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "config")

	bytes, err := json.MarshalIndent(gs, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path.Join(datadir, "genesis.json"), bytes, 0600)
}

func (d *IntArrayLedger) DumpLedger() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	lines := []string{}
	for i := int64(1); i <= d.height; i++ {
		if v, ok := d.blockContents[i]; ok {
			lines = append(lines, fmt.Sprintf("%d %s %s", i, v.GetHash().HashString(), v.String()))
		} else {
			lines = append(lines, fmt.Sprintf("%d EMPTY", i))
		}
	}
	err := files.WriteLines(lines, path.Join(datadir, "ledger.txt"))
	utilfuncs.PanicIfError(err, "dump ledger")
}

func (d *IntArrayLedger) loadLedger() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")

	if !files.FileExists(path.Join(datadir, "ledger.txt")) {
		return
	}
	lines, err := files.ReadLines(path.Join(datadir, "ledger.txt"))
	utilfuncs.PanicIfError(err, "read ledger")
	for _, line := range lines {
		slots := strings.Split(line, " ")
		height, err := strconv.Atoi(slots[0])
		utilfuncs.PanicIfError(err, "read line")
		hash := slots[1]
		numbers := slots[2]
		block := &IntArrayBlockContent{Values: []int{}}
		for _, number := range strings.Split(numbers, "_") {
			n, err := strconv.Atoi(number)
			utilfuncs.PanicIfError(err, "read number")
			block.Values = append(block.Values, n)
		}
		if block.GetHash().HashString() != hash {
			panic("hash not aligned ")
		}
		d.blockContents[int64(height)] = block
		d.height = math.BiggerInt64(d.height, int64(height))
		logrus.WithFields(
			logrus.Fields{
				"height": d.height,
				"hash":   block.GetHash().HashString(),
			}).Debug("loaded ledger")
	}
}

func (d *IntArrayLedger) CurrentHeight() int64 {
	return d.height
}

func (d *IntArrayLedger) CurrentCommittee() *consensus_interface.Committee {
	// currently no general election. always return committee in genesis
	return d.genesis.FirstCommittee
}
