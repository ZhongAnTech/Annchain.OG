package dummy

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/annchain/commongo/files"
	"github.com/annchain/commongo/format"
	"github.com/annchain/commongo/math"
	"github.com/annchain/commongo/utilfuncs"
	"github.com/latifrons/goffchan"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	LedgerFile             = "ledger.ledger"
	TempLedgerFile         = "ledger_temp.ledger"
	ConsensusStateFile     = "consensus_state.ledger"
	ConsensusCommitteeFile = "consensus_committee.ledger"
)

type IntArrayBlockContent struct {
	Height      int64
	Step        int    // Step is the action within current block
	PreviousSum int    // PreviousSum simulates the previous hash
	MySum       int    // MySum simulates the state root which is the ledger execution state.
	Submitter   int    // generator of this block
	Ts          string // generate time
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
	DataPath   string
	ConfigPath string
	Reporter   *soccerdash.Reporter

	height                                int64
	genesis                               *og_interface.Genesis
	confirmedBlockIdHeightMapping         map[string]int64
	confirmedBlockContents                map[int64]og_interface.BlockContent  // height-content
	allBlockContents                      map[string]og_interface.BlockContent // blockId-content
	consensusState                        *consensus_interface.ConsensusState
	unknownNeededEventSubscribers         []ogsyncer_interface.UnknownNeededEventSubscriber
	newLocalHeightUpdatedEventSubscribers []og_interface.NewLocalHeightUpdatedEventSubscriber
}

func (d *IntArrayLedger) Dump() {
	values := []*IntArrayBlockContent{}
	for _, v := range d.allBlockContents {
		values = append(values, v.(*IntArrayBlockContent))
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Height < values[j].Height
	})

	for _, value := range values {
		fmt.Printf("%d=%s\n", value.Height, value)

	}
	fmt.Printf("Total: %d records\n", len(values))
}

func (d *IntArrayLedger) InitDefault() {
	d.confirmedBlockIdHeightMapping = make(map[string]int64)
	d.confirmedBlockContents = make(map[int64]og_interface.BlockContent)
	d.allBlockContents = make(map[string]og_interface.BlockContent)
}

func (d *IntArrayLedger) GetConsensusState() *consensus_interface.ConsensusState {
	if d.consensusState == nil {
		logrus.Fatal("ledger consensus state is not inited")
	}
	return d.consensusState
}

func (d *IntArrayLedger) Speculate(prevBlockId string, block *consensus_interface.Block) (executionResult consensus_interface.ExecutionResult) {
	fmt.Println("Speculate Prev block:", prevBlockId)
	fmt.Println("Speculate Curr block:", block.Id)
	fmt.Println(block.String())
	_, ok := d.allBlockContents[prevBlockId]

	if !ok {
		logrus.WithField("prevBlockId", prevBlockId).Warn("prev block not found")
		return
		// wait until sync is done.
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
	d.KnowBlock(newBlock)
	d.SaveLedger()

	logrus.WithField("block", newBlock).Info("speculated new block")
	d.Reporter.Report("lastSpeculateHeight", newBlock.Height, false)

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
	d.ConfirmBlock(block)
}

// KnowBlock just knows the existance of the block.
// Block may not be confirmed.
func (d *IntArrayLedger) KnowBlock(block og_interface.BlockContent) {
	logrus.WithField("blockId", block.GetHash().HashString()).Debug("know this block")
	d.allBlockContents[block.GetHash().HashString()] = block
}

// ConfirmBlock add the block to the permanent ledger.
func (d *IntArrayLedger) ConfirmBlock(block og_interface.BlockContent) {
	if _, ok := d.confirmedBlockContents[block.GetHeight()]; ok {
		logrus.WithField("height", block.GetHeight()).Debug("block already added")
		return
	}
	id := block.GetHash().HashString()
	d.confirmedBlockContents[block.GetHeight()] = block
	d.confirmedBlockIdHeightMapping[id] = block.GetHeight()
	d.allBlockContents[id] = block
	// update height
	d.updateHeight(block)
	d.SaveLedger()
	d.Reporter.Report("lastCommit", fmt.Sprintf("%d %s", block.GetHeight(), block.GetHash().HashString()), false)
	logrus.WithField("block", block.GetHeight()).Warn("confirmed block")
}

func (d *IntArrayLedger) updateHeight(block og_interface.BlockContent) {
	// only update if block height = height + 1 so that we won't jump height
	// check if the block is in the cache. if the block exists, update height
	for {
		if d.height+1 == block.GetHeight() {
			d.height = math.BiggerInt64(block.GetHeight(), d.height)
			break
		} else {
			if _, ok := d.confirmedBlockContents[d.height+1]; ok {
				d.height = d.height + 1
				continue // there may be more to resolve
			} else {
				// no more to resolve, break
				break
			}
		}
	}
}

func (d *IntArrayLedger) GetBlock(height int64) og_interface.BlockContent {
	return d.confirmedBlockContents[height]
}

func (d *IntArrayLedger) AddRandomBlock(height int64) {
	previousBlock := d.GetBlock(height - 1).(*IntArrayBlockContent)
	step := rand.Intn(50)
	d.ConfirmBlock(&IntArrayBlockContent{
		Height:      height,
		Step:        step,
		PreviousSum: previousBlock.MySum,
		MySum:       step + previousBlock.MySum,
		Submitter:   0,
		Ts:          format.FormatTimeToStandard(time.Now()),
	})
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
	err = d.LoadTempLedger()
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

func (d *IntArrayLedger) ProvideGenesis() *IntArrayBlockContent {
	return &IntArrayBlockContent{
		Height:      0,
		Step:        0,
		PreviousSum: 0,
		MySum:       0,
		Submitter:   0,
		Ts:          "2019-02-02 14:29:00", // birth time of latifrons's little cute son
	}
}

// Ledger init
func (d *IntArrayLedger) InitLedgerGenesis() {
	logrus.Info("Initializing Ledger genesis")
	d.height = 0
	d.ConfirmBlock(d.ProvideGenesis())
}

func (d *IntArrayLedger) SaveLedger() {
	lines := []string{}
	for i := int64(0); i <= d.height; i++ {
		if v, ok := d.confirmedBlockContents[i]; ok {
			lines = append(lines, fmt.Sprintf("%d %s %s", i, v.GetHash().HashString(), v.String()))
		} else {
			lines = append(lines, fmt.Sprintf("%d EMPTY", i))
		}
	}
	err := files.WriteLines(lines, path.Join(d.DataPath, LedgerFile))
	utilfuncs.PanicIfError(err, "dump ledger")

	lines = []string{}
	// dump pending blocks
	for hash, block := range d.allBlockContents {
		_, ok := d.confirmedBlockIdHeightMapping[hash]
		if ok {
			// dumped
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s", hash, block.String()))
	}
	err = files.WriteLines(lines, path.Join(d.DataPath, TempLedgerFile))
	utilfuncs.PanicIfError(err, "dump ledger")
}

func (d *IntArrayLedger) LoadTempLedger() (err error) {
	if !files.FileExists(path.Join(d.DataPath, TempLedgerFile)) {
		err = errors.New("ledger file not exists: " + path.Join(d.DataPath, TempLedgerFile))
		return
	}
	lines, err := files.ReadLines(path.Join(d.DataPath, TempLedgerFile))
	if err != nil {
		return
	}

	for _, line := range lines {
		slots := strings.SplitN(line, " ", 3)
		hash := slots[0]
		blockString := slots[1]
		block := &IntArrayBlockContent{}
		block.FromString(blockString)

		if block.GetHash().HashString() != hash {
			return errors.New("hash not aligned")
		}
		d.KnowBlock(block)
		logrus.WithFields(
			logrus.Fields{
				"height": d.height,
				"hash":   block.GetHash().HashString(),
			}).Debug("loaded temp ledger")
	}
	return
}

func (d *IntArrayLedger) LoadLedger() (err error) {
	if !files.FileExists(path.Join(d.DataPath, LedgerFile)) {
		err = errors.New("ledger file not exists: " + path.Join(d.DataPath, LedgerFile))
		return
	}
	lines, err := files.ReadLines(path.Join(d.DataPath, LedgerFile))
	if err != nil {
		return
	}

	for _, line := range lines {
		slots := strings.SplitN(line, " ", 3)
		height, err := strconv.Atoi(slots[0])
		utilfuncs.PanicIfError(err, "read line")
		hash := slots[1]
		blockString := slots[2]
		block := &IntArrayBlockContent{}
		block.FromString(blockString)

		if block.GetHash().HashString() != hash {
			return errors.New("hash not aligned")
		}
		if block.Height != int64(height) {
			return errors.New("height not aligned")
		}
		d.ConfirmBlock(block)
		logrus.WithFields(
			logrus.Fields{
				"height": d.height,
				"hash":   block.GetHash().HashString(),
			}).Debug("loaded ledger")
	}
	// in case empty ledger
	if d.height == 0 {
		d.InitLedgerGenesis()
	}
	return
}

// Consensus Committee init
func (d *IntArrayLedger) InitConsensusCommitteeGenesis() {
	// init from config file.
	byteContent, err := ioutil.ReadFile(path.Join(d.ConfigPath, "consensus_genesis.json"))
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

	bytes, err := json.MarshalIndent(gs, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path.Join(d.DataPath, ConsensusCommitteeFile), bytes, 0600)
}

func (d *IntArrayLedger) LoadConsensusCommittee() (err error) {
	byteContent, err := ioutil.ReadFile(path.Join(d.DataPath, ConsensusCommitteeFile))
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
	// get hash of the first block
	genesis := d.ProvideGenesis()
	hash := genesis.GetHash().HashString()

	// generate high qc and genesis
	d.consensusState = &consensus_interface.ConsensusState{
		LastVoteRound:  0,
		PreferredRound: 0,
		HighQC: &consensus_interface.QC{
			VoteData: consensus_interface.VoteInfo{
				Id:               hash,
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
	logrus.WithField("state", consensusState).Trace("saving consensus state")
	bytes, err := json.MarshalIndent(consensusState, "", "    ")
	if err != nil {
		logrus.WithError(err).Fatal("dump consensus state")
	}

	err = ioutil.WriteFile(path.Join(d.DataPath, ConsensusStateFile), bytes, 0644)
	if err != nil {
		logrus.WithError(err).Fatal("save consensus state")
	}
}

func (d *IntArrayLedger) LoadConsensusState() (err error) {
	byteContent, err := ioutil.ReadFile(path.Join(d.DataPath, ConsensusStateFile))

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

func (d *IntArrayLedger) AddSubscriberUnknownNeededEvent(sub ogsyncer_interface.UnknownNeededEventSubscriber) {
	d.unknownNeededEventSubscribers = append(d.unknownNeededEventSubscribers, sub)
}

func (n *IntArrayLedger) notifyUnknownNeededEvent(event ogsyncer_interface.Unknown) {
	for _, subscriber := range n.unknownNeededEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.UnknownNeededEventChannel(), event, "notifyUnknownNeededEvent"+subscriber.Name()).C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

func (d *IntArrayLedger) AddSubscriberNewLocalHeightUpdatedEventSubscriber(sub og_interface.NewLocalHeightUpdatedEventSubscriber) {
	d.newLocalHeightUpdatedEventSubscribers = append(d.newLocalHeightUpdatedEventSubscribers, sub)
}

func (n *IntArrayLedger) notifyNewLocalHeightUpdatedEvent(event ogsyncer_interface.Unknown) {
	for _, subscriber := range n.newLocalHeightUpdatedEventSubscribers {
		<-goffchan.NewTimeoutSenderShort(subscriber.NewLocalHeightUpdatedChannel(), event, "notifyNewLocalHeightUpdatedEvent").C
		//subscriber.NewOutgoingMessageEventChannel() <- event
	}
}

type IntArrayProposalGenerator struct {
	Ledger    *IntArrayLedger
	rander    *rand.Rand
	BlockTime time.Duration
	MyId      int
}

func (i *IntArrayProposalGenerator) InitDefault() {
	i.rander = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (i IntArrayProposalGenerator) GenerateProposal(context *consensus_interface.ProposalContext) *consensus_interface.ContentProposal {
	var b og_interface.BlockContent = nil
	var ok bool
	if b, ok = i.Ledger.allBlockContents[context.HighQC.VoteData.Id]; !ok {
		logrus.WithField("blockId", context.HighQC.VoteData.Id).Warn("prev block not found in ledger. Propose further")
	}
	if b == nil {
		if b, ok = i.Ledger.allBlockContents[context.HighQC.VoteData.ParentId]; !ok {
			logrus.WithField("blockId", context.HighQC.VoteData.ParentId).Warn("parent block not found in ledger. Propose further")
		}
	}
	if b == nil {
		if b, ok = i.Ledger.allBlockContents[context.HighQC.VoteData.GrandParentId]; !ok {
			logrus.WithField("blockId", context.HighQC.VoteData.GrandParentId).Warn("gparent block not found in ledger.")
		}
	}
	if b == nil {
		logrus.Warn("i cannot propose since the previous blocks is still missing.")
		i.Ledger.Dump()
		return nil
	}

	previousBlock := b.(*IntArrayBlockContent)

	//previousBlock2 := i.Ledger.confirmedBlockContents[i.Ledger.height].(*IntArrayBlockContent)

	//if *previousBlock != *previousBlock2{
	//	fmt.Println(previousBlock)
	//	fmt.Println(previousBlock2)
	//	panic("not aligned")
	//}

	v := i.rander.Intn(100)
	newBlock := &IntArrayBlockContent{
		Height:      previousBlock.Height + 1,
		Step:        v,
		PreviousSum: previousBlock.MySum,
		MySum:       previousBlock.MySum + v,
		Submitter:   i.MyId,
		Ts:          format.FormatTimeToStandard(time.Now()),
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
	time.Sleep(i.BlockTime)
	i.Ledger.Reporter.Report("myProposal", proposal.String(), false)
	return proposal
}

func (i *IntArrayProposalGenerator) GenerateProposalAsync(context *consensus_interface.ProposalContext, callback func(*consensus_interface.ContentProposal)) {
	go func(context *consensus_interface.ProposalContext, callback func(*consensus_interface.ContentProposal)) {
		logrus.WithField("context", context).Info("start generating proposal")
		proposal := i.GenerateProposal(context)
		logrus.WithField("proposal", proposal).Info("proposal generated")
		callback(proposal)
	}(context, callback)
}
