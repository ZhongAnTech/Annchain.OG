package consensus_interface

// OgLedgerAccount represents a full account of a user.
type ConsensusAccount interface {
	Id() string
	//PubKey() crypto.PubKey
}

type CommitteeMember struct {
	PeerIndex        int              // order of peer in the committee
	MemberId         string           // peer identifier. current use address
	ConsensusAccount ConsensusAccount // account public key to verify messages
}

type Committee struct {
	Peers   []*CommitteeMember
	Version int
}

type ProposalContext struct {
	CurrentRound int64
	HighQC       *QC
	TC           *TC
}

type VerifyResult struct {
	Ok bool
}

type ExecuteResult struct {
	Ok        bool
	ExecuteId string
}

type ConsensusState struct {
	LastVoteRound  int64
	PreferredRound int64
}

type ConsensusAccountProvider interface {
	ProvideAccount() (ConsensusAccount, error)
	Generate() (account ConsensusAccount, err error)
	Load() (account ConsensusAccount, err error)
	Save() (err error)
}

type ProposalContextProvider interface {
	GetProposalContext() *ProposalContext
}

// ProposalGenerator provides a proposal whenever needed
type ProposalGenerator interface {
	GenerateProposal(context *ProposalContext) *ContentProposal
	GenerateProposalAsync(context *ProposalContext)
}

type ProposalVerifier interface {
	VerifyProposal(proposal *ContentProposal) *VerifyResult
	VerifyProposalAsync(proposal *ContentProposal)
}

type ProposalExecutor interface {
	ExecuteProposal(block *Block)
	ExecuteProposalAsync(block *Block)
}

type CommitteeProvider interface {
	InitCommittee(version int, peers []CommitteeMember, myAccount ConsensusAccount)
	GetVersion() int
	GetAllMemberPeedIds() []string
	GetAllMembers() []CommitteeMember
	GetMyPeerId() string
	GetMyPeerIndex() int
	GetLeaderPeerId(round int64) string
	GetPeerIndex(id string) (index int, err error)
	GetThreshold() int
	AmILeader(round int64) bool
	AmIIn() bool
	IsIn(id string) bool
}
type ConsensusSigner interface {
	Sign(msg []byte, account ConsensusAccount) Signature
}

type SignatureCollector interface {
	GetThreshold() int
	GetCurrentCount() int
	GetSignature(index int) (v Signature, ok bool)
	GetJointSignature() JointSignature
	Collected() bool
	Collect(sig Signature, index int)
	Has(index int) bool
}

type Ledger interface {
	// Speculate applies cmds speculatively
	Speculate(prevBlockId string, blockId string, cmds string) (executeStateId string)
	// GetState finds the pending state for the given blockId or nil if not present
	GetState(blockId string) (stateId string)
	// Commit commits the pending prefix of the given blockId and prune other branches
	Commit(blockId string)
	GetHighQC() *QC
	SetHighQC(qc *QC)
	SaveConsensusState(*ConsensusState)
	CurrentHeight() int64
	CurrentCommittee() *Committee
	//LoadConsensusState() *ConsensusState
}

type Hasher interface {
	Hash(s string) string
}

//type PendingTreeOrganizer interface {
//	//
//	Commit(blockId string)
//}
