package crypto

import (
	"encoding/binary"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/encrypt/ecies"
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/group/edwards25519"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/poc/extra25519"
	"github.com/annchain/OG/types"
)

type CryptoType int8

const (
	CryptoTypeEd25519 CryptoType = iota
	CryptoTypeSecp256k1
)

type PrivateKey struct {
	Type  CryptoType
	Bytes []byte
}

type PublicKey struct {
	Type  CryptoType
	Bytes []byte
}

type Signature struct {
	Type  CryptoType
	Bytes []byte
}

func PrivateKeyFromBytes(typev CryptoType, bytes []byte) PrivateKey {
	return PrivateKey{Type: typev, Bytes: bytes}
}
func PublicKeyFromBytes(typev CryptoType, bytes []byte) PublicKey {
	return PublicKey{Type: typev, Bytes: bytes}
}
func SignatureFromBytes(typev CryptoType, bytes []byte) Signature {
	return Signature{Type: typev, Bytes: bytes}
}

func PrivateKeyFromString(value string) (priv PrivateKey, err error) {
	bytes, err := hexutil.Decode(value)
	if err != nil {
		return
	}
	priv = PrivateKey{
		Type:  CryptoType(bytes[0]),
		Bytes: bytes[1:],
	}
	return
}

func PublicKeyFromString(value string) (pub PublicKey, err error) {
	bytes, err := hexutil.Decode(value)
	if err != nil {
		return
	}
	pub = PublicKey{
		Type:  CryptoType(bytes[0]),
		Bytes: bytes[1:],
	}
	return
}

func (k *PrivateKey) String() string {
	var bytes []byte
	bytes = append(bytes, byte(k.Type))
	bytes = append(bytes, k.Bytes...)
	return hexutil.Encode(bytes)
}

func (p *PrivateKey) PublicKey() *PublicKey {
	s := NewSigner(p.Type)
	pub := s.PubKey(*p)
	return &pub
}

func (p *PublicKey) Encrypt(m []byte) (ct []byte, err error) {
	s := NewSigner(p.Type)
	return s.Encrypt(*p, m)
}

func (p *PrivateKey) Decrypt(ct []byte) (m []byte, err error) {
	s := NewSigner(p.Type)
	return s.Decrypt(*p, ct)
}

type KyberEd22519PrivKey struct {
	PrivateKey kyber.Scalar
	Suit       *edwards25519.SuiteEd25519
}

func (p *KyberEd22519PrivKey) Decrypt(cipherText []byte) (m []byte, err error) {
	return ecies.Decrypt(p.Suit, p.PrivateKey, cipherText, p.Suit.Hash)
}

func (p *PrivateKey) ToKyberEd25519PrivKey() *KyberEd22519PrivKey {
	var edPrivKey [32]byte
	var curvPrivKey [64]byte
	copy(curvPrivKey[:], p.Bytes[:64])
	extra25519.PrivateKeyToCurve25519(&edPrivKey, &curvPrivKey)
	privateKey, err := edwards25519.UnmarshalBinaryScalar(edPrivKey[:32])
	suite := edwards25519.NewBlakeSHA256Ed25519()
	if err != nil {
		panic(err)
	}
	return &KyberEd22519PrivKey{
		PrivateKey: privateKey,
		Suit:       suite,
	}
}

func (p *PublicKey) Address() types.Address {
	s := NewSigner(p.Type)
	return s.Address(*p)
}

func (p *PublicKey) String() string {
	var bytes []byte
	bytes = append(bytes, byte(p.Type))
	bytes = append(bytes, p.Bytes...)
	return hexutil.Encode(bytes)
}

func NewSigner(cryptoType CryptoType) Signer {
	if cryptoType == CryptoTypeEd25519 {
		return &SignerEd25519{}
	} else if cryptoType == CryptoTypeSecp256k1 {
		return &SignerSecp256k1{}
	}
	return nil
}

func (c CryptoType) String() string {
	if c == CryptoTypeEd25519 {
		return "ed25519"
	} else if c == CryptoTypeSecp256k1 {
		return "secp256k1"
	}
	return "unknown"
}

// CreateAddress creates an ethereum address given the bytes and the nonce
func CreateAddress(b types.Address, nonce uint64) types.Address {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, nonce)
	return types.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], bs)[12:])
}

// CreateAddress2 creates an ethereum address given the address bytes, initial
// contract code hash and a salt.
func CreateAddress2(b types.Address, salt [32]byte, inithash []byte) types.Address {
	return types.BytesToAddress(Keccak256([]byte{0xff}, b.ToBytes()[:], salt[:], inithash)[12:])
}
