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

type IntArrayBlockContent struct {
	Step        int
	PreviousSum int
	MySum       int
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

type IntArrayLedger struct {
	height         int64
	genesis        *og_interface.Genesis
	blockContents  map[int64]og_interface.BlockContent // height-content
	highQC         *consensus_interface.QC
	consensusState *consensus_interface.ConsensusState
}

func (d *IntArrayLedger) Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string) {
	panic("implement me")
}

func (d *IntArrayLedger) GetState(blockId string) (stateId string) {
	panic("implement me")
}

func (d *IntArrayLedger) Commit(blockId string) {

}

func (d *IntArrayLedger) GetHighQC() *consensus_interface.QC {
	return d.highQC
}
func (d *IntArrayLedger) SetHighQC(qc *consensus_interface.QC) {
	d.highQC = qc
}

func (d *IntArrayLedger) SaveConsensusState(state *consensus_interface.ConsensusState) {
	bytes, err := json.Marshal(state)
	if err != nil {
		logrus.WithError(err).Fatal("dump consensus state")
	}

	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	err = ioutil.WriteFile(path.Join(datadir, "consensus.json"), bytes, 0644)
	if err != nil {
		logrus.WithError(err).Fatal("save consensus state")
	}
}

func (d *IntArrayLedger) loadConsensusState() (err error) {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	byteContent, err := ioutil.ReadFile(path.Join(datadir, "consensus.json"))

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

func (d *IntArrayLedger) AddBlock(height int64, block og_interface.BlockContent) {
	d.blockContents[height] = block
	d.height = math.BiggerInt64(height, d.height)
}

func (d *IntArrayLedger) GetBlock(height int64) og_interface.BlockContent {
	return d.blockContents[height]
}

func (d *IntArrayLedger) AddRandomBlock(height int64) {
	previousBlock := d.GetBlock(height - 1).(*IntArrayBlockContent)
	step := rand.Intn(50)
	d.AddBlock(height, &IntArrayBlockContent{
		Step:        step,
		PreviousSum: previousBlock.MySum,
		MySum:       step + previousBlock.MySum,
	})
}

func (d *IntArrayLedger) InitDefault() {
	d.blockContents = make(map[int64]og_interface.BlockContent)
}

// StaticSetup supposely will load ledger from disk.
func (d *IntArrayLedger) StaticSetup() {
	reInit := false
	// load from local storage first.
	err := d.loadLedger()
	if err != nil {
		reInit = true
	}
	err = d.loadConsensusState()
	if err != nil {
		reInit = true
	}
	// if error, init ledger and consensus
	if reInit {
		d.InitGenesisLedger()
	}

	d.height = 0
	rootHash := &og_interface.Hash32{}
	rootHash.FromHexNoError("0x00")

	// check if ledger exists.
	d.InitGenesisLedger()

	//d.LoadConsensusGenesis()
	//for i := 1; i < 10000; i ++{
	//	d.AddRandomBlock(int64(i))
	//}

	//d.AddRandomBlock(2)
	//d.AddRandomBlock(3)
	//d.AddRandomBlock(4)
	//d.AddRandomBlock(5)
	//d.dumpLedger()

}

// InitGenesisLedger generate genesis predefined and store it in the database file
func (d *IntArrayLedger) InitGenesisLedger() {
	// generate high qc and genesis
	d.highQC = &consensus_interface.QC{
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
	}
	d.height = 0
	d.blockContents[0] = &IntArrayBlockContent{
		Step:        0,
		PreviousSum: 0,
		MySum:       0,
	}
	err := d.LoadConsensusGenesis()
	if err != nil {
		// consensus genesis template not exists, cannot do further operation
		logrus.WithError(err).Fatal("failed to load consensus genesis")
	}
	// write to ledger
	d.dumpLedger()
	d.DumpConsensusStatus()
}

func (d *IntArrayLedger) LoadConsensusGenesis() (err error) {
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

	// convert to genesis
	//hash := &og_interface.Hash32{}
	//err = hash.FromHex(gs.RootSequencerHash)
	//utilfuncs.PanicIfError(err, "root seq hash")

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
	return
}

func (d *IntArrayLedger) DumpConsensusGenesis() {

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

	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "config")

	bytes, err := json.MarshalIndent(gs, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path.Join(datadir, "consensus_genesis.json"), bytes, 0600)
}

func (d *IntArrayLedger) DumpConsensusStatus() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	j, err := json.MarshalIndent(d.highQC, "", "    ")
	utilfuncs.PanicIfError(err, "marshal high qc")
	err = ioutil.WriteFile(path.Join(datadir, "consensus_status.json"), j, 0644)
	utilfuncs.PanicIfError(err, "dump consensus")
}

func (d *IntArrayLedger) LoadConsensusStatus() error {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	content, err := ioutil.ReadFile(path.Join(datadir, "consensus_status.json"))
	if err != nil {
		return err
	}

	d.highQC = &consensus_interface.QC{}

	err = json.Unmarshal(content, d.highQC)
	if err != nil {
		return err
	}
	return nil
}

func (d *IntArrayLedger) dumpLedger() {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")
	lines := []string{}
	for i := int64(0); i <= d.height; i++ {
		if v, ok := d.blockContents[i]; ok {
			lines = append(lines, fmt.Sprintf("%d %s %s", i, v.GetHash().HashString(), v.String()))
		} else {
			lines = append(lines, fmt.Sprintf("%d EMPTY", i))
		}
	}
	err := files.WriteLines(lines, path.Join(datadir, "ledger.txt"))

	utilfuncs.PanicIfError(err, "dump ledger")
}

func (d *IntArrayLedger) loadLedger() (err error) {
	datadir := files.FixPrefixPath(viper.GetString("rootdir"), "data")

	if !files.FileExists(path.Join(datadir, "ledger.txt")) {
		err = errors.New("ledger file not exists: " + path.Join(datadir, "ledger.txt"))
		return
	}
	lines, err := files.ReadLines(path.Join(datadir, "ledger.txt"))
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
		d.blockContents[int64(height)] = block
		d.height = math.BiggerInt64(d.height, int64(height))
		logrus.WithFields(
			logrus.Fields{
				"height": d.height,
				"hash":   block.GetHash().HashString(),
			}).Debug("loaded ledger")
	}
	return
}

func (d *IntArrayLedger) CurrentHeight() int64 {
	return d.height
}

func (d *IntArrayLedger) CurrentCommittee() *consensus_interface.Committee {
	// currently no general election. always return committee in genesis
	return d.genesis.FirstCommittee
}

type IntArrayProposalGenerator struct {
	Ledger *IntArrayLedger
	rander *rand.Rand
}

func (i *IntArrayProposalGenerator) InitDefault() {
	i.rander = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (i IntArrayProposalGenerator) GenerateProposal(context *consensus_interface.ProposalContext) *consensus_interface.ContentProposal {
	previousBlock := i.Ledger.blockContents[context.CurrentRound-1].(*IntArrayBlockContent)
	v := i.rander.Intn(100)
	newBlock := &IntArrayBlockContent{
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
