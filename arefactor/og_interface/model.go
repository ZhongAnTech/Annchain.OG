package og_interface

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	cryptopb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

type CryptoType int

const CryptoTypeRSA = CryptoType(cryptopb.KeyType_RSA)
const CryptoTypeEd25519 = CryptoType(cryptopb.KeyType_Ed25519)
const CryptoTypeSecp256k1 = CryptoType(cryptopb.KeyType_Secp256k1)
const CryptoTypeECDSA = CryptoType(cryptopb.KeyType_ECDSA)

// OgLedgerAccount represents a full account of a user.
type OgLedgerAccount struct {
	PublicKey  crypto.PubKey
	PrivateKey crypto.PrivKey
	Address    Address
}

type OgTx struct {
	Hash        Hash
	ParentsHash []Hash
	MineNonce   uint64
	From        Address
	To          Address
	Value       string // bigint
	TokenId     int32
	PublicKey   []byte
	Data        []byte
	Signature   []byte
}

type OgSequencer struct {
}
