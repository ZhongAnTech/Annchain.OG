package og

import (
	"bytes"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
	"github.com/annchain/OG/common/crypto"
	"golang.org/x/crypto/ripemd160"
)

func AddressFromPublicKey(p *ogcrypto_interface.PublicKey) types.Address {
	switch p.Type {
	case ogcrypto_interface.CryptoTypeSecp256k1:
		return AddressFromPublicKeySecp256K1(p)
	case ogcrypto_interface.CryptoTypeEd25519:
		return AddressFromPublicKeyEd25519(p)
	default:
		panic("unknown public key type")
	}
}

func AddressFromPublicKeyEd25519(pubKey *ogcrypto_interface.PublicKey) types.Address {
	var w bytes.Buffer
	w.Write([]byte{byte(pubKey.Type)})
	w.Write(pubKey.KeyBytes)
	hasher := ripemd160.New()
	hasher.Write(w.Bytes())
	result := hasher.Sum(nil)
	return types.BytesToAddress(result)
}

func AddressFromPublicKeySecp256K1(pubKey *ogcrypto_interface.PublicKey) types.Address {
	return types.BytesToAddress(crypto.Keccak256((pubKey.KeyBytes)[1:])[12:])
}
