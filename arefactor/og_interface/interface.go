package og_interface

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	"io"
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

type NewHeightDetectedEventSubscriber interface {
	Name() string
	NewHeightDetectedEventChannel() chan *NewHeightDetectedEvent
}

type LedgerAccountHolder interface {
	ProvideAccount() (*OgLedgerAccount, error)
	Generate(src io.Reader) (account *OgLedgerAccount, err error)
	Load() (account *OgLedgerAccount, err error)
	Save() (err error)
}

type AddressConverter interface {
	AddressFromAccount(account *OgLedgerAccount) (addr Address, err error)
}

type PrivateGenerator interface {
	GeneratePair(typ int, src io.Reader) (privKey crypto.PrivKey, pubKey crypto.PubKey, err error)
}
