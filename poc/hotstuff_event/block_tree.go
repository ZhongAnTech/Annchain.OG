package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

type Block struct {
	Round    int    // the round that generated this proposal
	Payload  string // proposed transactions
	ParentQC *QC    // qc for parent block
	Id       string // unique digest of round, payload and parent_qc.id
}

func (b Block) String() string {
	return fmt.Sprintf("[Block round=%d payload=%s parentQC=%s id=%s]", b.Round, b.Payload, b.ParentQC, b.Id)
}

type VoteInfo struct {
	Id               string // Id of the block
	Round            int    // round of the block
	ParentId         string // Id of the parent
	ParentRound      int    // round of the parent
	GrandParentId    string // Id of the grandParent
	GrandParentRound int    // round of the grandParent
	ExecStateId      string // speculated execution state
}

func (i VoteInfo) String() string {
	return fmt.Sprintf("<%s %d> <%s %d> <%s %d> %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

func (i VoteInfo) GetHashContent() string {
	return fmt.Sprintf("%s %d %s %d %s %d %s", i.Id, i.Round, i.ParentId, i.ParentRound, i.GrandParentId, i.GrandParentRound, i.ExecStateId)
}

type LedgerCommitInfo struct {
	CommitStateId string // nil if no commit happens when this vote is aggregated to QC. Usually the merkle root
	VoteInfoHash  string // hash of VoteMsg.voteInfo
}

func (l LedgerCommitInfo) String() string {
	return fmt.Sprintf("[LedgerCommitInfo: commitStateId %s VoteInfoHash %s]", l.CommitStateId, l.VoteInfoHash)
}

func (l LedgerCommitInfo) GetHashContent() string {
	return fmt.Sprintf("%s %s", l.CommitStateId, l.VoteInfoHash)
}

type VoteMsg struct {
	VoteInfo         VoteInfo
	LedgerCommitInfo LedgerCommitInfo
	Sender           int
	Signature        Signature
}

type SignatureCollector struct {
	Signatures map[int]Signature
}

func (s *SignatureCollector) InitDefault() {
	s.Signatures = make(map[int]Signature)
}

func (s *SignatureCollector) Collect(signature Signature) {
	s.Signatures[signature.PartnerId] = signature
}

func (s *SignatureCollector) Count() int {
	return len(s.Signatures)
}

func (s *SignatureCollector) AllSignatures() []Signature {
	sigs := make([]Signature, len(s.Signatures))
	i := 0
	for _, value := range s.Signatures {
		sigs[i] = value
	}
	return sigs
}

func (s *SignatureCollector) Has(key int) bool {
	_, ok := s.Signatures[key]
	return ok
}

type BlockTree struct {
	Logger    *logrus.Logger
	Ledger    *Ledger
	PaceMaker *PaceMaker
	F         int

	pendingBlkTree PendingBlockTree              // tree of blocks pending commitment
	pendingVotes   map[string]SignatureCollector // collected votes per block indexed by their LedgerInfo hash
	highQC         *QC                           // highest known QC
}

func (t *BlockTree) InitGenesisOrLatest() {
	t.highQC = &QC{
		VoteInfo: VoteInfo{
			Id:               "genesis",
			Round:            0,
			ParentId:         "",
			ParentRound:      0,
			GrandParentId:    "",
			GrandParentRound: 0,
			ExecStateId:      "genesisstate",
		},
		LedgerCommitInfo: LedgerCommitInfo{
			CommitStateId: "genesisstate",
			VoteInfoHash:  "votehash",
		},
		Signatures: nil,
	}
	//t.pendingBlkTree.Add(&Block{
	//	Round:    0,
	//	Payload:  "genesispayload",
	//	ParentQC: nil,
	//	Id:       "genesis",
	//})
}

func (t *BlockTree) InitDefault() {
	t.pendingBlkTree.InitDefault()
	t.pendingBlkTree.Logger = t.Logger
	t.pendingVotes = make(map[string]SignatureCollector)
}

func (t *BlockTree) ProcessVote(vote *ContentVote, signature Signature) {
	voteIndex := Hash(vote.LedgerCommitInfo.GetHashContent())
	if _, ok := t.pendingVotes[voteIndex]; !ok {
		collector := SignatureCollector{}
		collector.InitDefault()
		t.pendingVotes[voteIndex] = collector
	}
	collector := t.pendingVotes[voteIndex]
	collector.Collect(signature)
	if collector.Count() == 2*t.F+1 {
		qc := &QC{
			VoteInfo:         vote.VoteInfo,
			LedgerCommitInfo: vote.LedgerCommitInfo,
			Signatures:       collector.AllSignatures(),
		}
		t.PaceMaker.AdvanceRound(qc, "vote qc got")
		if qc.VoteInfo.Round > t.highQC.VoteInfo.Round {
			t.highQC = qc
		}

	}
}

func (t *BlockTree) ProcessCommit(id string) {
	t.Ledger.Commit(id)
	t.pendingBlkTree.Commit(id)
}

func (t *BlockTree) ExecuteAndInsert(p *Block) {
	// it is a proposal message
	executeStateId := t.Ledger.Speculate(p.ParentQC.VoteInfo.Id, p.Id, p.Payload)
	fmt.Println(executeStateId)
	t.pendingBlkTree.Add(p)
	if p.ParentQC.VoteInfo.Round > t.highQC.VoteInfo.Round {
		t.highQC = p.ParentQC
	}

}

func (t *BlockTree) GenerateProposal(currentRound int, payload string) *ContentProposal {
	return &ContentProposal{
		Proposal: Block{
			Round:    currentRound,
			Payload:  payload,
			ParentQC: t.highQC,
			Id:       Hash(fmt.Sprintf("%d %s %s", currentRound, payload, t.highQC)),
		},
	}
}
