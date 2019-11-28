package annsensus

import (
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"time"
)

// TermIdProvider provide Dkg term that will be changed every term switching.
// TermIdProvider maintains historic Peer info that can be retrieved by height.
// Once a new term is started, the term info will be sent through TermChangeEventChannel
type TermIdProvider interface {
	// HeightTerm maps height to dkg term
	HeightTerm(height uint64) (termId uint32)
	CurrentTerm() (termId uint32)
	GetTermChangeEventChannel() chan ConsensusContextProvider
}

// HistoricalTermsHolder saves all historical term handlers(bft, dkg) and term info.
// In case of some slow messages are coming.
type HistoricalTermsHolder interface {
	GetTermByHeight(heightInfoCarrier HeightInfoCarrier) (msgTerm *TermCollection, err error)
	GetTermById(u uint32) (msgTerm *TermCollection, err error)
	SetTerm(u uint32, composer *TermCollection)
	DebugMyId() int
}

// ConsensusContextProvider provides not only term info but also the character the node play in this term.
type ConsensusContextProvider interface {
	GetTerm() *term.Term
	GetMyBftId() int
	GetMyPartSec() dkg.PartSec
	GetBlockTime() time.Duration
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
	AdaptAnnsensusPeer(AnnsensusPeer) (bft.PeerInfo, error)

	AdaptBftMessage(outgoingMsg bft.BftMessage) (AnnsensusMessage, error)
	AdaptBftPeer(bftPeer bft.PeerInfo) (AnnsensusPeer, error)
}

type DkgMessageAdapter interface {
	AdaptAnnsensusMessage(incomingMsg AnnsensusMessage) (dkg.DkgMessage, error)
	AdaptAnnsensusPeer(AnnsensusPeer) (dkg.PeerInfo, error)

	AdaptDkgMessage(outgoingMsg dkg.DkgMessage) (AnnsensusMessage, error)
	AdaptDkgPeer(dkgPeer dkg.PeerInfo) (AnnsensusPeer, error)
}

type AnnsensusMessage interface {
	GetType() AnnsensusMessageType
	GetData() []byte
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
