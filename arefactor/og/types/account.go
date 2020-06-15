package types

import (
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	cryptopb "github.com/libp2p/go-libp2p-core/crypto/pb"
)

type CryptoType int

const CryptoTypeRSA = CryptoType(cryptopb.KeyType_RSA)
const CryptoTypeEd25519 = CryptoType(cryptopb.KeyType_Ed25519)
const CryptoTypeSecp256k1 = CryptoType(cryptopb.KeyType_Secp256k1)
const CryptoTypeECDSA = CryptoType(cryptopb.KeyType_ECDSA)

// OgAccount represents a full account of a user.
type OgAccount struct {
	PublicKey  crypto.PubKey
	PrivateKey crypto.PrivKey
	Address    og_interface.Address
}
