package og

//import (
//	"bytes"
//	"github.com/annchain/OG/arefactor/og/types"
//	"github.com/annchain/OG/arefactor/og_interface"
//	"github.com/libp2p/go-libp2p-core/crypto"
//	"golang.org/x/crypto/ripemd160"
//)
//
//func AddressFromPublicKey(p crypto.PubKey) og_interface.Address {
//	switch p.Type {
//	case ogcrypto_interface.CryptoTypeSecp256k1:
//		return AddressFromPublicKeySecp256K1(p)
//	case ogcrypto_interface.CryptoTypeEd25519:
//		return AddressFromPublicKeyEd25519(p)
//	default:
//		panic("unknown public key type")
//	}
//}
//
//func AddressFromPublicKeyEd25519(pubKey *ogcrypto_interface.PublicKey) og_interface.Address {
//	var w bytes.Buffer
//	w.Write([]byte{byte(pubKey.Type)})
//	w.Write(pubKey.KeyBytes)
//	hasher := ripemd160.New()
//	hasher.Write(w.Bytes())
//	result := hasher.Sum(nil)
//
//	address := &og_interface.Address20{}
//	address.FromBytes(result)
//	return address
//}
//
//func AddressFromPublicKeySecp256K1(pubKey *ogcrypto_interface.PublicKey) og_interface.Address {
//	address := &og_interface.Address20{}
//	address.FromBytes(ogcrypto2.Keccak256((pubKey.KeyBytes)[1:])[12:])
//	return address
//}
