package og

import (
	"bytes"
	"encoding/json"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
	"io"
	"io/ioutil"
)

type LocalAccountProvider struct {
	CryptoType       types.CryptoType
	AddressConverter AddressConverter
	Account          *types.OgAccount
}

type AccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
}

func (l *LocalAccountProvider) Load(filePath string) (account *types.OgAccount, err error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	als := &AccountLocalStorage{}
	err = json.Unmarshal(bytes, als)
	if err != nil {
		return
	}
	privKeyBytes, err := hexutil.FromHex(als.PrivKey)
	if err != nil {
		return
	}

	unmarshaller := crypto.PrivKeyUnmarshallers[pb.KeyType(als.CryptoType)]
	privKey, err := unmarshaller(privKeyBytes)
	if err != nil {
		return
	}

	account = &types.OgAccount{
		PublicKey:  privKey.GetPublic(),
		PrivateKey: privKey,
	}
	account.Address, err = l.AddressConverter.AddressFromPubKey(account.PublicKey)
	return
}

func (l *LocalAccountProvider) Save(filePath string, account *types.OgAccount) (err error) {
	pubKeyBytes, err := account.PublicKey.Raw()
	if err != nil {
		return
	}
	privKeyBytes, err := account.PrivateKey.Raw()
	if err != nil {
		return
	}
	als := &AccountLocalStorage{
		CryptoType: int32(account.PublicKey.Type()),
		PubKey:     hexutil.ToHex(pubKeyBytes),
		PrivKey:    hexutil.ToHex(privKeyBytes),
	}

	bytes, err := json.MarshalIndent(als, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filePath, bytes, 0600)
	return

}

type AccountGenerator struct {
	AddressConverter AddressConverter
}

func (g *AccountGenerator) Generate(typ types.CryptoType, src io.Reader) (account *types.OgAccount, err error) {
	var privKey crypto.PrivKey
	var pubKey crypto.PubKey
	switch typ {
	case types.CryptoTypeRSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, src)
	case types.CryptoTypeEd25519:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Ed25519, 0, src)
	case types.CryptoTypeSecp256k1:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 0, src)
	case types.CryptoTypeECDSA:
		privKey, pubKey, err = crypto.GenerateKeyPairWithReader(crypto.ECDSA, 0, src)
	}
	if err != nil {
		return
	}
	addr, err := g.AddressConverter.AddressFromPubKey(pubKey)
	if err != nil {
		return
	}
	account = &types.OgAccount{
		PublicKey:  pubKey,
		PrivateKey: privKey,
		Address:    addr,
	}
	return

}

type AddressConverter interface {
	AddressFromPubKey(pubkey crypto.PubKey) (addr og_interface.Address, err error)
}

type OgAddressConverter struct {
}

func (o *OgAddressConverter) AddressFromPubKey(pubKey crypto.PubKey) (addr og_interface.Address, err error) {
	bytes, err := pubKey.Bytes()
	if err != nil {
		return
	}
	addr = &og_interface.Address20{}
	addr.FromBytes(Keccak256(bytes))
	return
}

func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

func (o *OgAddressConverter) AddressFromPubKey1(pubKey crypto.PubKey) (addr og_interface.Address, err error) {
	var w bytes.Buffer
	bytes, err := pubKey.Bytes()
	if err != nil {
		return
	}
	w.Write(bytes)
	hasher := ripemd160.New()
	hasher.Write(w.Bytes())
	result := hasher.Sum(nil)
	addr = &og_interface.Address20{}
	addr.FromBytes(result)
	return
}
