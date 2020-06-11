package og

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/common/files"
	"github.com/annchain/OG/arefactor/common/math"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/minio/sha256-simd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	CurrentCommittee() *Committee
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

	d.genesis = &Genesis{
		RootSequencerHash: rootHash,
		FirstCommittee: &Committee{
			Peers: []*PeerMember{
				{
					PeerId:    "16Uiu2HAmSPLF68qLtob31r3hYga7Qs84TPas9wNkanKQtKezjRne",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmKK8gG13RdesZenARuPmsYnXQH5iLad85eMoepRQS6Pi9",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmGWF1gqXsJ5oZzy213bQRh1aWqSRdiowRecNNH2DEqkYL",
					PublicKey: nil,
				},
				{
					PeerId:    "16Uiu2HAmTb7wFyTqjWTMrjdUNZXRUdXh4byQi6nBNGGnYe8vhdG3",
					PublicKey: nil,
				},
			},
			Version: 1,
		},
	}
	//d.AddRandomBlock(1)
	//d.AddRandomBlock(2)
	//d.AddRandomBlock(3)
	//d.AddRandomBlock(4)
	//d.AddRandomBlock(5)
	//d.dumpLedger()
	d.loadLedger()
}

func (d *IntArrayLedger) dumpLedger() {
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

func (d *IntArrayLedger) CurrentCommittee() *Committee {
	// currently no general election. always return committee in genesis
	return d.genesis.FirstCommittee
}
