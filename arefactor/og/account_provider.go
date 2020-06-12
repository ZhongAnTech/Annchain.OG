package og

import (
	crand "crypto/rand"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
	core "github.com/libp2p/go-libp2p-core/crypto"
)

type LocalAccountProvider struct {
	CryptoType ogcrypto_interface.CryptoType
	Account    *types.OgAccount
}

func (l *LocalAccountProvider) Generate(randomizer io.Reader) {
	//if randomizer == nil {
	//	randomizer = crand.Reader
	//}
	//
	//priv, _, err := core.GenerateKeyPairWithReader(core.Secp256k1, 0, randomizer)
	//if err != nil {
	//	return err
	//}
}
