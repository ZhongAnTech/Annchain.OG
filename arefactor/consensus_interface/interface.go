package consensus_interface

import (
	"github.com/libp2p/go-libp2p-core/crypto"
)

// OgLedgerAccount represents a full account of a user.
type ConsensusAccount struct {
	PublicKey  crypto.PubKey
	PrivateKey crypto.PrivKey
}

func (c *ConsensusAccount) Id() string {
	return ""
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
}

type VerifyResult struct {
	Ok bool
}

type ExecuteResult struct {
	Ok        bool
	ExecuteId string
}

type ConsensusState struct {
	LastVoteRound  int
	PreferredRound int
}

type ConsensusAccountProvider interface {
	ProvideAccount() (*ConsensusAccount, error)
	Generate() (account *ConsensusAccount, err error)
	Load() (err error)
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
	GetLeaderPeerId(round int) string
	GetPeerIndex(id string) (index int, err error)
	GetThreshold() int
	AmILeader(round int) bool
	AmIIn() bool
	IsIn(id string) bool
}
type Signer interface {
	Sign(msg []byte, privateKey crypto.PrivKey) Signature
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
