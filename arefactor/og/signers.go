package og

//import (
//	"encoding/binary"
//	"github.com/annchain/OG/og_interface"
//	ogcrypto2 "github.com/annchain/OG/deprecated/ogcrypto"
//	"github.com/annchain/OG/deprecated/ogcrypto_interface"
//)
//
//// CreateAddress creates an ethereum address given the bytes and the nonce
//func CreateAddress(b og_interface.Address, nonce uint64) og_interface.Address {
//	bs := make([]byte, 8)
//	binary.LittleEndian.PutUint64(bs, nonce)
//	address := &og_interface.Address20{}
//	address.FromBytes(ogcrypto2.Keccak256([]byte{0xff}, b.Bytes(), bs)[12:])
//	return address
//}
//
//// CreateAddress2 creates an ethereum address given the address bytes, initial
//// contract code hash and a salt.
//func CreateAddress2(b og_interface.Address, salt [32]byte, inithash []byte) og_interface.Address {
//	address := &og_interface.Address20{}
//	address.FromBytes(ogcrypto2.Keccak256([]byte{0xff}, b.Bytes(), salt[:], inithash)[12:])
//	return address
//}
//
//func NewSigner(cryptoType ogcrypto_interface.CryptoType) ogcrypto_interface.ISigner {
//	if cryptoType == ogcrypto_interface.CryptoTypeEd25519 {
//		return &ogcrypto2.SignerEd25519{}
//	} else if cryptoType == ogcrypto_interface.CryptoTypeSecp256k1 {
//		return &ogcrypto2.SignerSecp256k1{}
//	}
//	return nil
//}
