package og

import (
	"encoding/binary"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
)

// CreateAddress creates an ethereum address given the bytes and the nonce
func CreateAddress(b og_interface.Address, nonce uint64) og_interface.Address {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	address := &og_interface.Address20{}
	address.FromBytes(ogcrypto.Keccak256([]byte{0xff}, b.Bytes(), bs)[12:])
	return address
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b og_interface.Address, salt [32]byte, inithash []byte) og_interface.Address {
	address := &og_interface.Address20{}
	address.FromBytes(ogcrypto.Keccak256([]byte{0xff}, b.Bytes(), salt[:], inithash)[12:])
	return address
}

func NewSigner(cryptoType ogcrypto_interface.CryptoType) ogcrypto_interface.ISigner {
	if cryptoType == ogcrypto_interface.CryptoTypeEd25519 {
		return &ogcrypto.SignerEd25519{}
	} else if cryptoType == ogcrypto_interface.CryptoTypeSecp256k1 {
		return &ogcrypto.SignerSecp256k1{}
	}
	return nil
}
