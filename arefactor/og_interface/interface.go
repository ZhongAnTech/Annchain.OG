package og_interface

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type PeerJoinedEvent struct {
	PeerId string
}
type PeerLeftEvent struct {
	PeerId string
}

type SequencerReceivedEvent struct {
}

type TxReceivedEvent struct {
}

type IntsReceivedEvent struct {
	Ints ogsyncer_interface.MessageContentInt
	From string
}

type NodeInfoProvider interface {
	CurrentHeight() int64
	GetNetworkId() string
}

type NewHeightDetectedEvent struct {
	Height int64
	PeerId string
}

type NewLocalHeightUpdatedEvent struct {
	Height int64
}
type ResourceGotEvent struct {
	ResourceType int
	Resource     interface{}
	From         string
}

type AccountHolder interface {
	ProvidePrivateKey(createIfMissing bool) ([]byte, error)
}
type LedgerSigner interface {
	Sign(msg []byte, account OgLedgerAccount) []byte
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
}
