package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"time"
)

// TermIdProvider provide Dkg term that will be changed every term switching.
// TermIdProvider maintains historic Sender info that can be retrieved by height.
// Once a new term is started, the term info will be sent through TermChangeEventChannel
type TermIdProvider interface {
	// HeightTerm maps height to dkg term
	HeightTerm(height uint64) (termId uint32)
	CurrentTerm() (termId uint32)
	GetTermChangeEventChannel() chan ConsensusContext
}

// HistoricalTermsHolder saves all historical term handlers(bft, dkg) and term info.
// In case of some slow messages are coming.
type HistoricalTermsHolder interface {
	GetTermByHeight(heightInfoCarrier HeightInfoCarrier) (msgTerm *TermCollection, ok bool)
	GetTermById(termId uint32) (msgTerm *TermCollection, ok bool)
	SetTerm(termId uint32, composer *TermCollection)
	DebugMyId() int
}

// ConsensusContext provides not only term info but also the character the node play in this term.
type ConsensusContext interface {
	GetTerm() *term.Term
	GetMyBftId() int
	GetMyPartSec() dkg.PartSec
	GetBlockTime() time.Duration
}

type ConsensusContextProvider interface {
	GetConsensusContext(newTerm *term.Term) ConsensusContext
}

// HeightProvider is called when a height is needed.
// It is usually implemented by the chain info
type HeightProvider interface {
	CurrentHeight() uint64
}

//type SequencerProducer interface {
//	GenerateSequencer(issuer common.Address, height uint64, accountNonce uint64,
//		privateKey *crypto.PrivateKey, blsPubKey []byte) (seq *types.Sequencer, err error, genAgain bool)
//	ValidateSequencer(seq types.Sequencer) error
//}

// BftMessageAdapter converts messages.
// During the converting process there may be some validation and signing operations.
type BftMessageAdapter interface {
	AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (bft.BftMessage, error)
	AdaptAnnsensusPeer(AnnsensusPeer) (bft.BftPeer, error)

	AdaptBftMessage(outgoingMsg bft.BftMessage) (AnnsensusMessage, error)
	AdaptBftPeer(bftPeer bft.BftPeer) (AnnsensusPeer, error)
}

type DkgMessageAdapter interface {
	AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (dkg.DkgMessage, error)
	AdaptAnnsensusPeer(AnnsensusPeer) (dkg.DkgPeer, error)

	AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (AnnsensusMessage, error)
	AdaptDkgPeer(dkgPeer dkg.DkgPeer) (AnnsensusPeer, error)
}

type AnnsensusMessage interface {
	GetType() AnnsensusMessageType
	GetBytes() []byte
	String() string
}

type AnnsensusPeerCommunicatorOutgoing interface {
	Broadcast(msg AnnsensusMessage, peers []AnnsensusPeer)
	Unicast(msg AnnsensusMessage, peer AnnsensusPeer)
}

type AnnsensusPeerCommunicatorIncoming interface {
	GetPipeIn() chan *AnnsensusMessageEvent
	GetPipeOut() chan *AnnsensusMessageEvent
}

type AnnsensusMessageHandler interface {
	HandleMessage(msg AnnsensusMessage, peer AnnsensusPeer)
	//GetPipeIn() chan AnnsensusMessage
	//GetPipeOut() chan AnnsensusMessage
}
