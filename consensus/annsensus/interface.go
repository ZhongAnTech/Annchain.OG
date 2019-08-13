package annsensus

import (
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/types/tx_types"
)

// TermProvider provide Dkg term that will be changed every term switching.
// TermProvider maintains historic Peer info that can be retrieved by height.

type TermProvider interface {
	// HeightTerm maps height to dkg term
	HeightTerm(height uint64) (termId uint32)
	CurrentTerm() (termId uint32)
	// Peers returns all peers at given term
	Peers(termId uint32) []bft.PeerInfo
	GetTermChangeEventChannel() chan *term.Term
}

type AccountNonceProvider interface {
	GetNonce(account *account.Account) uint64
}

// ConsensusAccountProvider provides public key and private key for signing consensus messages
type ConsensusAccountProvider interface {
	Account() *account.Account
}

// HeightProvider is called when a height is needed
type HeightProvider interface {
	CurrentHeight() uint64
}

type SequencerProducer interface {
	GenerateSequencer(issuer common.Address, height uint64, accountNonce uint64,
		privateKey *crypto.PrivateKey, blsPubKey []byte) (seq *tx_types.Sequencer, err error, genAgain bool)
	ValidateSequencer(seq tx_types.Sequencer) error
}
