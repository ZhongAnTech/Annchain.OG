package hotstuff_event

import (
	"fmt"
	"github.com/latifrons/soccerdash"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type SignatureCollector struct {
	Signatures map[int]Signature
	mu         sync.RWMutex
}

func (s *SignatureCollector) InitDefault() {
	s.Signatures = make(map[int]Signature)
}

func (s *SignatureCollector) Collect(signature Signature) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Signatures[signature.PartnerIndex] = signature
}

func (s *SignatureCollector) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Signatures)
}

func (s *SignatureCollector) AllSignatures() []Signature {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sigs := make([]Signature, len(s.Signatures))
	i := 0
	for _, signature := range s.Signatures {
		sigs[i] = signature
		i += 1
	}
	return sigs
}

func (s *SignatureCollector) Has(key int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Signatures[key]
	return ok
}

type BlockTree struct {
	Logger    *logrus.Logger
	Ledger    *Ledger
	PaceMaker *PaceMaker
	F         int
	MyIdIndex int
	Report    *soccerdash.Reporter

	pendingBlkTree PendingBlockTree              // tree of blocks pending commitment
	pendingVotes   map[string]SignatureCollector // collected votes per block indexed by their LedgerInfo hash
	highQC         *QC                           // highest known QC
}

func (t *BlockTree) InitGenesisOrLatest() {
	t.updateHighQC(&QC{

		VoteData: VoteInfo{
			Id:               "genesis",
			Round:            0,
			ParentId:         "genesis-1",
			ParentRound:      0,
			GrandParentId:    "genesis-2",
			GrandParentRound: 0,
			ExecStateId:      "genesis-state",
		},
		Signatures: nil,
	})
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
		MyId:   t.MyIdIndex,
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
	logrus.WithField("sigs", collector.Signatures).WithField("sig", signature).Debug("signature got one")
	if collector.Count() == 2*t.F+1 {
		t.Logger.WithField("vote", vote).Info("votes collected")
		qc := &QC{
			VoteData:   vote.VoteInfo, // TODO: check if the voteinfo is aligned
			Signatures: collector.AllSignatures(),
		}
		if qc.VoteData.Round > t.highQC.VoteData.Round {
			t.updateHighQC(qc)
		}
		t.PaceMaker.AdvanceRound(qc, nil, "vote qc got")

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
	executeStateId := t.Ledger.Speculate(p.ParentQC.VoteData.Id, p.Id, p.Payload)
	t.Logger.WithFields(logrus.Fields{
		"round":          p.Round,
		"total":          t.Ledger.cache[p.Id].total,
		"blockid":        p.Id,
		"blockContent":   p.Payload,
		"executeStateId": executeStateId,
	}).Info("Block Executed")
	t.Report.Report("blockId", p.Id, false)
	t.Report.Report("executeStateId", executeStateId, false)
	t.pendingBlkTree.Add(p)
	if p.ParentQC.VoteData.Round > t.highQC.VoteData.Round {
		t.Logger.WithField("old", t.highQC).WithField("new", p.ParentQC).Info("highQC updated")
		t.updateHighQC(p.ParentQC)
	} else {
		t.Logger.WithField("old", t.highQC).WithField("new", p.ParentQC).Warn("highQC is not updated")
	}

}

func (t *BlockTree) GenerateProposal(currentRound int, payload string) *ContentProposal {
	time.Sleep(time.Second * 1)
	//time.Sleep(time.Millisecond * 2)
	return &ContentProposal{
		Proposal: Block{
			Round:    currentRound,
			Payload:  payload,
			ParentQC: t.highQC,
			Id:       Hash(fmt.Sprintf("%d %s %s", currentRound, payload, t.highQC)),
		},
		TC: t.PaceMaker.lastTC,
	}
}

func (t *BlockTree) updateHighQC(qc *QC) {
	t.highQC = qc
	t.Report.Report("t.highQC", t.highQC.VoteData.Round, false)
}
