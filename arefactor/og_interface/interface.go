package og_interface

import (
	"github.com/annchain/OG/arefactor/consensus_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type PeerJoinedEventArg struct {
	PeerId string
}
type PeerLeftEventArg struct {
	PeerId string
}

type NodeInfoProvider interface {
	CurrentHeight() int64
	GetNetworkId() string
}

type NewHeightDetectedEventArg struct {
	Height int64
	PeerId string
}

type NewLocalHeightUpdatedEventArg struct {
	Height int64
}
type ResourceGotEvent struct {
	ResourceType int
	Resource     interface{}
	From         string
}
type NewHeightBlockSyncedEventArg struct {
	Height int64
}

type NewBlockProducedEventArg struct {
	Block BlockContent
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
	GetBlockByHash(hash string) BlockContent
	ConfirmBlock(block BlockContent)
}
