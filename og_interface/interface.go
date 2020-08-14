package og_interface

import (
	"github.com/annchain/OG/consensus_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type PeerJoinedEvent struct {
	PeerId string
}
type PeerLeftEvent struct {
	PeerId string
}

type PeerJoinedEventSubscriber interface {
	EventChannelPeerJoined() chan *PeerJoinedEvent
}

type PeerLeftEventSubscriber interface {
	EventChannelPeerLeft() chan *PeerLeftEvent
}

type NodeInfoProvider interface {
	CurrentHeight() int64
	GetNetworkId() string
}

type NewHeightDetectedEvent struct {
	Height int64
	PeerId string
}

type AccountHolder interface {
	ProvidePrivateKey(createIfMissing bool) ([]byte, error)
}
type LedgerSigner interface {
	Sign(msg []byte, account OgLedgerAccount) []byte
}

type NewHeightDetectedEventSubscriber interface {
	Name() string
	NewHeightDetectedEventChannel() chan *NewHeightDetectedEvent
}

type LedgerAccountProvider interface {
	ProvideAccount() (*OgLedgerAccount, error)
	Generate() (account *OgLedgerAccount, err error)
	Load() (account *OgLedgerAccount, err error)
	Save() (err error)
}

type AddressConverter interface {
	AddressFromAccount(account *OgLedgerAccount) (addr Address, err error)
}

type PrivateGenerator interface {
	GeneratePair(typ int) (privKey crypto.PrivKey, pubKey crypto.PubKey, err error)
}

type BlockContent interface {
	GetType() BlockContentType
	String() string
	FromString(string)
	GetHash() Hash
	GetHeight() int64
}

type Ledger interface {
	CurrentHeight() int64
	CurrentCommittee() *consensus_interface.Committee
	GetBlock(height int64) BlockContent
	ConfirmBlock(block BlockContent)
	GetResource(request ResourceRequest) Resource
}
