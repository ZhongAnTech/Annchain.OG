package transport_interface

import (
	"github.com/libp2p/go-libp2p-core/crypto"
	cryptopb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

type CryptoType int

const CryptoTypeRSA = CryptoType(cryptopb.KeyType_RSA)
const CryptoTypeEd25519 = CryptoType(cryptopb.KeyType_Ed25519)
const CryptoTypeSecp256k1 = CryptoType(cryptopb.KeyType_Secp256k1)
const CryptoTypeECDSA = CryptoType(cryptopb.KeyType_ECDSA)

// TransportAccount represents a full account of a user in p2p network.
type TransportAccount struct {
	PublicKey  crypto.PubKey
	PrivateKey crypto.PrivKey
	NodeId     string
}
