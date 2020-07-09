package og

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/crypto"
	"io"
)

type CachedPrivateGenerator struct {
	Refresh         bool
	Reader          io.Reader
	previousPrivKey crypto.PrivKey
	previousPubKey  crypto.PubKey
}

func (p *CachedPrivateGenerator) GeneratePair(typ int) (privKey crypto.PrivKey, pubKey crypto.PubKey, err error) {
	if p.Reader == nil {
		p.Reader = rand.Reader
	}

	if !p.Refresh && p.previousPrivKey != nil && p.previousPubKey != nil {
		return p.previousPrivKey, p.previousPubKey, nil
	}

	switch typ {
	case crypto.RSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, p.Reader)
	case crypto.Ed25519:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, p.Reader)
	case crypto.Secp256k1:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 0, p.Reader)
	case crypto.ECDSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.ECDSA, 0, p.Reader)
	}
	p.previousPrivKey = privKey
	p.previousPubKey = pubKey
	return
}
