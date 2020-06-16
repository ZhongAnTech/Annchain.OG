package og

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/crypto"
	"io"
)

type PrivateGenerator interface {
	GeneratePair(typ int, src io.Reader) (privKey crypto.PrivKey, pubKey crypto.PubKey, err error)
}

type DefaultPrivateGenerator struct {
	Reader io.Reader
}

func (p *DefaultPrivateGenerator) GeneratePair(typ int, givenRandomReader io.Reader) (privKey crypto.PrivKey, pubKey crypto.PubKey, err error) {
	reader := givenRandomReader
	if reader == nil {
		reader = p.Reader
	}
	if reader == nil {
		p.Reader = rand.Reader
		reader = p.Reader
	}

	switch typ {
	case crypto.RSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, reader)
	case crypto.Ed25519:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, reader)
	case crypto.Secp256k1:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 0, reader)
	case crypto.ECDSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.ECDSA, 0, reader)
	}
	return
}
