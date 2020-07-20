package dummy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/common/files"
	"github.com/annchain/OG/arefactor/common/math"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/minio/sha256-simd"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	LedgerFile             = "ledger.ledger"
	ConsensusStateFile     = "consensus_state.ledger"
	ConsensusCommitteeFile = "consensus_committee.ledger"
)

type IntArrayBlockContent struct {
	Height      int64
	Step        int // Step is the action within current block
	PreviousSum int // PreviousSum simulates the previous hash
	MySum       int // MySum simulates the state root which is the ledger execution state.
}

func (i *IntArrayBlockContent) FromString(s string) {
	err := json.Unmarshal([]byte(s), i)
	utilfuncs.PanicIfError(err, "ledger unmarshal")
}

func (i *IntArrayBlockContent) GetType() og_interface.BlockContentType {
	return og_interface.BlockContentTypeInt
}

func (i *IntArrayBlockContent) String() string {
	// use json to dump
	bytes, err := json.Marshal(i)
	utilfuncs.PanicIfError(err, "ledger marshal")
	return string(bytes)
}

func (i *IntArrayBlockContent) GetHash() og_interface.Hash {
	s, err := json.Marshal(i.String())
	utilfuncs.PanicIfError(err, "get block hash")
	sum := sha256.Sum256(s)
	h := &og_interface.Hash32{}
	h.FromBytes(sum[:])
	return h
}

func (i *IntArrayBlockContent) GetHeight() int64 {
	return i.Height
}

type IntArrayLedger struct {
	height                        int64
	genesis                       *og_interface.Genesis
	confirmedBlockIdHeightMapping map[string]int64
	confirmedBlockContents        map[int64]og_interface.BlockContent  // height-content
	allBlockContents              map[string]og_interface.BlockContent // height-content
	consensusState                *consensus_interface.ConsensusState
}

func (d *IntArrayLedger) GetConsensusState() *consensus_interface.ConsensusState {
	if d.consensusState == nil {
		logrus.Fatal("ledger consensus state is not inited")
	}
	return d.consensusState
}

func (d *IntArrayLedger) Speculate(prevBlockId string, block *consensus_interface.Block) (executionResult consensus_interface.ExecutionResult) {
	fmt.Println(prevBlockId)
	fmt.Println(block.Id)
	fmt.Println(block.String())
	_, ok := d.allBlockContents[prevBlockId]

	if !ok {
		logrus.WithField("prevBlockId", prevBlockId).Fatal("prev block not found")
	}

	newBlock := &IntArrayBlockContent{}
	newBlock.FromString(block.Payload)

	// simulate execution. MySum is equivalent to state root and Step is equivalent to txs
	newBlock.MySum = newBlock.PreviousSum + newBlock.Step

	// my hash should be the same as given
	if block.Id != newBlock.GetHash().HashString() {
		logrus.WithField("myhash", newBlock.GetHash().HashString()).
			WithField("given", block.Id).
			Fatal("myhash is not aligned with given hash")
	}
	d.allBlockContents[block.Id] = newBlock

	return consensus_interface.ExecutionResult{
		BlockId:        newBlock.GetHash().HashString(),
		ExecuteStateId: fmt.Sprintf("%d", newBlock.MySum),
		Err:            nil,
	}
}

func (d *IntArrayLedger) GetState(blockId string) (stateId string) {
	block, ok := d.allBlockContents[blockId]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", block.(*IntArrayBlockContent).MySum)
}

func (d *IntArrayLedger) Commit(blockId string) {
	block, ok := d.allBlockContents[blockId]
	if !ok {
		logrus.WithField("blockId", blockId).Warn("blockId not found")
		return
	}
	d.AddBlock(block)
}

func (d *IntArrayLedger) AddBlock(block og_interface.BlockContent) {
	if _, ok := d.confirmedBlockContents[block.GetHeight()]; ok {
		logrus.WithField("height", block.GetHeight()).Fatal("block already added")
	}
	d.confirmedBlockContents[block.GetHeight()] = block
	d.height = math.BiggerInt64(block.GetHeight(), d.height)
}

func (d *IntArrayLedger) GetBlock(height int64) og_interface.BlockContent {
	return d.confirmedBlockContents[height]
}

func (d *IntArrayLedger) AddRandomBlock(height int64) {
	previousBlock := d.GetBlock(height - 1).(*IntArrayBlockContent)
	step := rand.Intn(50)
	d.AddBlock(&IntArrayBlockContent{
		Height:      height,
		Step:        step,
		PreviousSum: previousBlock.MySum,
		MySum:       step + previousBlock.MySum,
	})
}

func (d *IntArrayLedger) InitDefault() {
	d.confirmedBlockContents = make(map[int64]og_interface.BlockContent)
}

// StaticSetup supposely will load ledger from disk.
func (d *IntArrayLedger) StaticSetup() {
	// load all, if any failed, re-init all.
	reInit := false
	// load from local storage first.
	// if error, init ledger and consensus
	var err error
	err = d.LoadLedger()
	if err != nil {
		reInit = true
	}
	err = d.LoadConsensusState()
	if err != nil {
		reInit = true
	}
	err = d.LoadConsensusCommittee()
	if err != nil {
		reInit = true
	}
	if reInit {
		d.InitLedgerGenesis()
		d.InitConsensusStateGenesis()
		d.InitConsensusCommitteeGenesis()
		d.SaveLedger()
		d.SaveConsensusState(d.consensusState)
		d.SaveConsensusCommittee()
	}

	//rootHash := &og_interface.Hash32{}
	//rootHash.FromHexNoError("0x00")

	// check if ledger exists.
	//d.InitLedgerGenesis()

	//d.LoadConsensusCommittee()
	//for i := 1; i < 10000; i ++{
	//	d.AddRandomBlock(int64(i))
	//}

	//d.AddRandomBlock(2)
	//d.AddRandomBlock(3)
	//d.AddRandomBlock(4)
	//d.AddRandomBlock(5)
	//d.SaveLedger()

}

// Ledger init
func (d *IntArrayLedger) InitLedgerGenesis() {
	logrus.Info("Initializing Ledger genesis")
	d.height = 0
	d.confirmedBlockContents[0] = &IntArrayBlockContent{
		Height:      0,
		Step:        0,
		PreviousSum: 0,
		MySum:       0,
	}
}

func (d *IntArrayLedger) SaveLedger() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	lines := []string{}
	for i := int64(0); i <= d.height; i++ {
		if v, ok := d.confirmedBlockContents[i]; ok {
			lines = append(lines, fmt.Sprintf("%d %s %s", i, v.GetHash().HashString(), v.String()))
		} else {
			lines = append(lines, fmt.Sprintf("%d EMPTY", i))
		}
	}
	err := files.WriteLines(lines, path.Join(datadir, LedgerFile))
	utilfuncs.PanicIfError(err, "dump ledger")
}

func (d *IntArrayLedger) LoadLedger() (err error) {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")

	if !files.FileExists(path.Join(datadir, LedgerFile)) {
		err = errors.New("ledger file not exists: " + path.Join(datadir, LedgerFile))
		return
	}
	lines, err := files.ReadLines(path.Join(datadir, LedgerFile))
	if err != nil {
		return
	}

	for _, line := range lines {
		slots := strings.Split(line, " ")
		height, err := strconv.Atoi(slots[0])
		utilfuncs.PanicIfError(err, "read line")
		hash := slots[1]
		blockString := slots[2]
		block := &IntArrayBlockContent{}
		block.FromString(blockString)

		if block.GetHash().HashString() != hash {
			return errors.New("hash not aligned")
		}
		d.confirmedBlockContents[int64(height)] = block
		d.height = math.BiggerInt64(d.height, int64(height))
		logrus.WithFields(
			logrus.Fields{
				"height": d.height,
				"hash":   block.GetHash().HashString(),
			}).Debug("loaded ledger")
	}
	return
}

// Consensus Committee init
func (d *IntArrayLedger) InitConsensusCommitteeGenesis() {
	// init from config file.
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "config")
	byteContent, err := ioutil.ReadFile(path.Join(datadir, "consensus_genesis.json"))
	if err != nil {
		return
	}

	gs := &og_interface.GenesisStore{}
	err = json.Unmarshal(byteContent, gs)
	if err != nil {
		return
	}

	d.applyGenesisStore(gs)
}

func (d *IntArrayLedger) SaveConsensusCommittee() {
	peers := []og_interface.CommitteeMemberStore{}
	for _, v := range d.genesis.FirstCommittee.Peers {
		//pubKeyBytes, err := v.ConsensusAccount.PublicKey.Raw()
		//utilfuncs.PanicIfError(err, "pubkey raw")
		peers = append(peers, og_interface.CommitteeMemberStore{
			PeerIndex:       v.PeerIndex,
			MemberId:        v.MemberId,
			TransportPeerId: v.TransportPeerId,
			//PublicKey: hexutil.ToHex(pubKeyBytes),
		})
	}

	gs := &og_interface.GenesisStore{
		//RootSequencerHash: d.genesis.RootSequencerHash.HashString(),
		FirstCommittee: og_interface.CommitteeStore{
			Version: d.genesis.FirstCommittee.Version,
			Peers:   peers,
		},
	}

	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")

	bytes, err := json.MarshalIndent(gs, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path.Join(datadir, ConsensusCommitteeFile), bytes, 0600)
}

func (d *IntArrayLedger) LoadConsensusCommittee() (err error) {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	byteContent, err := ioutil.ReadFile(path.Join(datadir, ConsensusCommitteeFile))
	if err != nil {
		return
	}

	gs := &og_interface.GenesisStore{}
	err = json.Unmarshal(byteContent, gs)
	if err != nil {
		return
	}

	d.applyGenesisStore(gs)
	return
}

// Consensus State init
func (d *IntArrayLedger) InitConsensusStateGenesis() {
	// generate high qc and genesis
	d.consensusState = &consensus_interface.ConsensusState{
		LastVoteRound:  0,
		PreferredRound: 0,
		HighQC: &consensus_interface.QC{
			VoteData: consensus_interface.VoteInfo{
				Id:               "",
				Round:            0,
				ParentId:         "",
				ParentRound:      0,
				GrandParentId:    "",
				GrandParentRound: 0,
				ExecStateId:      "",
			},
			JointSignature: nil,
		},
	}
	return
}
func (d *IntArrayLedger) SaveConsensusState(consensusState *consensus_interface.ConsensusState) {
	bytes, err := json.Marshal(consensusState)
	if err != nil {
		logrus.WithError(err).Fatal("dump consensus state")
	}

	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	err = ioutil.WriteFile(path.Join(datadir, ConsensusStateFile), bytes, 0644)
	if err != nil {
		logrus.WithError(err).Fatal("save consensus state")
	}
}

func (d *IntArrayLedger) LoadConsensusState() (err error) {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	byteContent, err := ioutil.ReadFile(path.Join(datadir, ConsensusStateFile))

	if err != nil {
		logrus.WithError(err).Debug("error on loading consensus state")
		return
	}
	cs := &consensus_interface.ConsensusState{}
	err = json.Unmarshal(byteContent, cs)
	if err != nil {
		logrus.WithError(err).Fatal("unmarshal consensus state")
		return
	}
	d.consensusState = cs
	return
}

func (d *IntArrayLedger) CurrentHeight() int64 {
	return d.height
}

func (d *IntArrayLedger) CurrentCommittee() *consensus_interface.Committee {
	// currently no general election. always return committee in genesis
	return d.genesis.FirstCommittee
}

func (d *IntArrayLedger) applyGenesisStore(gs *og_interface.GenesisStore) {
	peers := []*consensus_interface.CommitteeMember{}
	//unmarshaller := crypto.PubKeyUnmarshallers[pb.KeyType_Secp256k1]

	for _, v := range gs.FirstCommittee.Peers {
		//pubKeyBytes, err := hexutil.FromHex(v.PublicKey)
		//utilfuncs.PanicIfError(err, "pubkey")
		//pubKey, err := unmarshaller(pubKeyBytes)
		//utilfuncs.PanicIfError(err, "pubkey")

		peers = append(peers, &consensus_interface.CommitteeMember{
			PeerIndex:        v.PeerIndex,
			MemberId:         v.MemberId,
			TransportPeerId:  v.TransportPeerId,
			ConsensusAccount: nil,
		})
	}
	g := &og_interface.Genesis{
		//RootSequencerHash: hash,
		FirstCommittee: &consensus_interface.Committee{
			Peers:   peers,
			Version: gs.FirstCommittee.Version,
		},
	}
	d.genesis = g
}

type IntArrayProposalGenerator struct {
	Ledger *IntArrayLedger
	rander *rand.Rand
}

func (i *IntArrayProposalGenerator) InitDefault() {
	i.rander = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (i IntArrayProposalGenerator) GenerateProposal(context *consensus_interface.ProposalContext) *consensus_interface.ContentProposal {
	previousBlock := i.Ledger.confirmedBlockContents[context.CurrentRound-1].(*IntArrayBlockContent)
	v := i.rander.Intn(100)
	newBlock := &IntArrayBlockContent{
		Height:      i.Ledger.height + 1,
		Step:        v,
		PreviousSum: previousBlock.MySum,
		MySum:       previousBlock.MySum + v,
	}
	proposal := &consensus_interface.ContentProposal{
		Proposal: consensus_interface.Block{
			Round:    context.CurrentRound,
			Payload:  newBlock.String(),
			ParentQC: context.HighQC,
			Id:       newBlock.GetHash().HashString(),
		},
		TC: context.TC,
	}
	return proposal
}

func (i IntArrayProposalGenerator) GenerateProposalAsync(context *consensus_interface.ProposalContext) {
	panic("implement me")
}
