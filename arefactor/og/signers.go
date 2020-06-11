package og

import (
	"encoding/binary"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
)

// CreateAddress creates an ethereum address given the bytes and the nonce
func CreateAddress(b types.Address, nonce uint64) types.Address {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	return types.BytesToAddress(ogcrypto.Keccak256([]byte{0xff}, b[:], bs)[12:])
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b types.Address, salt [32]byte, inithash []byte) types.Address {
	return types.BytesToAddress(ogcrypto.Keccak256([]byte{0xff}, b[:], salt[:], inithash)[12:])
}

func NewSigner(cryptoType ogcrypto_interface.CryptoType) ogcrypto_interface.ISigner {
	if cryptoType == ogcrypto_interface.CryptoTypeEd25519 {
		return &ogcrypto.SignerEd25519{}
	} else if cryptoType == ogcrypto_interface.CryptoTypeSecp256k1 {
		return &ogcrypto.SignerSecp256k1{}
	}
	return nil
}
