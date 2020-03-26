package hotstuff_event

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

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
	MyId      int

	pendingBlkTree PendingBlockTree              // tree of blocks pending commitment
	pendingVotes   map[string]SignatureCollector // collected votes per block indexed by their LedgerInfo hash
	highQC         *QC                           // highest known QC
}

func (t *BlockTree) InitGenesisOrLatest() {
	t.highQC = &QC{
		VoteInfo: VoteInfo{
			Id:               "genesis",
			Round:            0,
			ParentId:         "genesis-1",
			ParentRound:      0,
			GrandParentId:    "genesis-2",
			GrandParentRound: 0,
			ExecStateId:      "genesisstate",
		},
		LedgerCommitInfo: LedgerCommitInfo{
			CommitStateId: "genesisstate",
			VoteInfoHash:  "votehash",
		},
		Signatures: nil,
	}
	t.Ledger.Speculate("", "genesis", "0")
	t.Ledger.Commit("genesis")
	//t.pendingBlkTree.Add(&Block{
	//	Round:    0,
	//	Payload:  "genesispayload",
	//	ParentQC: nil,
	//	Id:       "genesis",
	//})
}

func (t *BlockTree) InitDefault() {
	t.pendingBlkTree = PendingBlockTree{
		MyId:   t.MyId,
		Logger: t.Logger,
	}
	t.pendingBlkTree.InitDefault()
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
		t.Logger.WithField("vote", vote).Trace("votes collected")
		qc := &QC{
			VoteInfo:         vote.VoteInfo,
			LedgerCommitInfo: vote.LedgerCommitInfo,
			Signatures:       collector.AllSignatures(),
		}
		if qc.VoteInfo.Round > t.highQC.VoteInfo.Round {
			t.highQC = qc
		}
		t.PaceMaker.AdvanceRound(qc, "vote qc got")

	} else {
		t.Logger.WithField("vote", vote).WithField("now", collector.Count()).Trace("votes yet collected")
	}
}

func (t *BlockTree) ProcessCommit(id string) {
	t.Ledger.Commit(id)
	t.pendingBlkTree.Commit(id)
}

func (t *BlockTree) ExecuteAndInsert(p *Block) {
	// it is a proposal message
	executeStateId := t.Ledger.Speculate(p.ParentQC.VoteInfo.Id, p.Id, p.Payload)
	fmt.Printf("[%d] at [%d-%d] [%s] ExecuteState: %s\n", t.MyId, p.Round, t.Ledger.cache[p.Id].total, p.Id, executeStateId)
	t.pendingBlkTree.Add(p)
	if p.ParentQC.VoteInfo.Round > t.highQC.VoteInfo.Round {
		t.highQC = p.ParentQC
	}

}

func (t *BlockTree) GenerateProposal(currentRound int, payload string) *ContentProposal {
	//time.Sleep(time.Second * 5)
	//time.Sleep(time.Millisecond * 2)
	return &ContentProposal{
		Proposal: Block{
			Round:    currentRound,
			Payload:  payload,
			ParentQC: t.highQC,
			Id:       Hash(fmt.Sprintf("%d %s %s", currentRound, payload, t.highQC)),
		},
	}
}
