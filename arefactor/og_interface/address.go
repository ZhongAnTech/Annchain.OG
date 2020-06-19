package og_interface

import (
	"crypto/sha256"
	"encoding/binary"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	ogCrypto "github.com/annchain/OG/arefactor/ogcrypto"
	"github.com/annchain/OG/common/math"
	"math/big"
	"math/rand"
)

const (
	Address20Length = 20
)

type Address20 [Address20Length]byte

func (a *Address20) AddressKey() AddressKey {
	return AddressKey(a[:])
}

func (a *Address20) AddressShortString() string {
	return hexutil.ToHex(a[:10])
}

func (a *Address20) AddressString() string {
	return hexutil.ToHex(a.Bytes())
}

func (a *Address20) Bytes() []byte {
	return a[:]
}

func (a *Address20) Hex() string {
	return hexutil.ToHex(a[:])
}

func (a *Address20) Length() int {
	return Address20Length
}

func (a *Address20) FromBytes(b []byte) {
	copy(a[:], b)
}

func (a *Address20) FromHex(s string) (err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	a.FromBytes(bytes)
	return
}

func (a *Address20) FromHexNoError(s string) {
	err := a.FromHex(s)
	utilfuncs.PanicIfError(err, "HexToAddress20")
}

func (a *Address20) Cmp(another FixLengthBytes) int {
	return BytesCmp(a, another)
}

func BytesToAddress20(b []byte) *Address20 {
	a := &Address20{}
	a.FromBytes(b)
	return a
}

func HexToAddress20(hex string) (*Address20, error) {
	a := &Address20{}
	err := a.FromHex(hex)
	return a, err
}

func BigToAddress20(v *big.Int) *Address20 {
	a := &Address20{}
	a.FromBytes(v.Bytes())
	return a
}

func RandomAddress20() *Address20 {
	v := math.NewBigInt(rand.Int63())
	adr := BigToAddress20(v.Value)
	h := sha256.New()
	data := []byte("abcd8342804fhddhfhisfdyr89")
	h.Write(adr[:])
	h.Write(data)
	sum := h.Sum(nil)
	adr.FromBytes(sum[:20])
	return adr
}

// CreateAddress creates an ethereum address given the bytes and the nonce
func CreateAddress20(b Address20, nonce uint64) Address20 {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	return BytesToAddress20(ogCrypto.Keccak256([]byte{0xff}, b.Bytes()[:], bs)[12:])
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress20_2(b Address20, salt Hash32, inithash []byte) Address20 {
	return BytesToAddress20(ogCrypto.Keccak256([]byte{0xff}, b.Bytes()[:], salt[:], inithash)[12:])
}
