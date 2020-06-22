package consensus_interface

import "github.com/libp2p/go-libp2p-core/crypto"

type Committee struct {
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
	ExecuteProposal(proposal *ContentProposal)
	ExecuteProposalAsync(proposal *ContentProposal)
}

type CommitteeProvider interface {
	GetSessionId() int
	GetAllMemberPeedIds() []string
	GetMyPeerId() string
	GetMyPeerIndex() int
	GetLeaderPeerId(round int) string
	GetPeerIndex(id string) (index int, err error)
}

type AccountHolder interface {
	ProvidePrivateKey(createIfMissing bool) (crypto.PrivKey, error)
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
}

type Hasher interface {
	Hash(s string) string
}

//type PendingTreeOrganizer interface {
//	//
//	Commit(blockId string)
//}
