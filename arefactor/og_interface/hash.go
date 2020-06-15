package og_interface

import (
	"crypto/sha256"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/annchain/OG/common/math"
	"math/big"
	"math/rand"
)

const (
	Hash32Length = 32
)

type Hash32 [Hash32Length]byte

func (a *Hash32) HashShortString() string {
	return hexutil.ToHex(a[:10])
}

func (a *Hash32) HashString() string {
	return hexutil.ToHex(a.Bytes())
}

func (a *Hash32) Bytes() []byte {
	return a[:]
}

func (a *Hash32) Hex() string {
	return hexutil.ToHex(a[:])
}

func (a *Hash32) Length() int {
	return Hash32Length
}

func (a *Hash32) FromBytes(b []byte) {
	copy(a[:], b)
}

func (a *Hash32) FromHex(s string) (err error) {
	bytes, err := hexutil.FromHex(s)
	if err != nil {
		return
	}
	a.FromBytes(bytes)
	return
}

func (a *Hash32) FromHexNoError(s string) {
	err := a.FromHex(s)
	utilfuncs.PanicIfError(err, "HexToHash32")
}

func (a *Hash32) Cmp(another FixLengthBytes) int {
	return BytesCmp(a, another)
}

func BytesToHash32(b []byte) *Hash32 {
	h := &Hash32{}
	h.FromBytes(b)
	return h
}

func HexToHash32(hex string) (*Hash32, error) {
	h := &Hash32{}
	err := h.FromHex(hex)
	return h, err
}

func BigToHash32(v *big.Int) *Hash32 {
	h := &Hash32{}
	h.FromBytes(v.Bytes())
	return h
}

func RandomHash32() *Hash32 {
	v := math.NewBigInt(rand.Int63())
	sh := BigToHash32(v.Value)
	h := sha256.New()
	data := []byte("456544546fhjsiodiruheswer8ih")
	h.Write(sh[:])
	h.Write(data)
	sum := h.Sum(nil)
	sh.FromBytes(sum)
	return sh
}
