package annsensus

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/types/msg"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"time"
)

// TermProvider provide Dkg term that will be changed every term switching.
// TermProvider maintains historic Peer info that can be retrieved by height.

type TermProvider interface {
	// HeightTerm maps height to dkg term
	HeightTerm(height uint64) (termId uint32)
	CurrentTerm() (termId uint32)
	// Peers returns all peers at given term
	Peers(termId uint32) ([]bft.PeerInfo, error)
	GetTermChangeEventChannel() chan ConsensusContextProvider
}

type TermHolder interface {
	GetTermCollection(heightInfoCarrier HeightInfoCarrier) (msgTerm *TermCollection, err error)
	SetTerm(u uint32, composer *TermCollection)
}

type ConsensusContextProvider interface {
	GetTermId() uint32
	GetNbParticipants() int
	GetThreshold() int
	GetMyBftId() int
	GetBlockTime() time.Duration
	GetSuite() *bn256.Suite
	GetAllPartPubs() []dkg.PartPub
	GetMyPartSec() dkg.PartSec
}

// HeightProvider is called when a height is needed
type HeightProvider interface {
	CurrentHeight() uint64
}

type SequencerProducer interface {
	GenerateSequencer(issuer common.Address, height uint64, accountNonce uint64,
		privateKey *crypto.PrivateKey, blsPubKey []byte) (seq *protocol_message.Sequencer, err error, genAgain bool)
	ValidateSequencer(seq protocol_message.Sequencer) error
}

// BftMessageAdapter converts messages.
// During the converting process there may be some validation and signing operations.
type BftMessageAdapter interface {
	AdaptOgMessage(incomingMsg msg.TransportableMessage) (bft.BftMessage, error)
	AdaptBftMessage(outgoingMsg bft.BftMessage) (msg.TransportableMessage, error)
}

type DkgMessageAdapter interface {
	AdaptOgMessage(incomingMsg msg.TransportableMessage) (dkg.DkgMessage, error)
	AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (msg.TransportableMessage, error)
}
