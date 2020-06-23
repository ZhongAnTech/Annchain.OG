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

type LedgerAccountHolder struct {
	CryptoType       types.CryptoType
	AddressConverter AddressConverter
	Account          *types.OgLedgerAccount
}

type LedgerAccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
}

func (l *LedgerAccountHolder) Load(filePath string) (account *types.OgLedgerAccount, err error) {
	byteContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}
	als := &LedgerAccountLocalStorage{}
	err = json.Unmarshal(byteContent, als)
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

	account = &types.OgLedgerAccount{
		PublicKey:  privKey.GetPublic(),
		PrivateKey: privKey,
	}
	account.Address, err = l.AddressConverter.AddressFromPubKey(account.PublicKey)
	return
}

func (l *LedgerAccountHolder) Save(filePath string, account *types.OgLedgerAccount) (err error) {
	pubKeyBytes, err := account.PublicKey.Raw()
	if err != nil {
		return
	}
	privKeyBytes, err := account.PrivateKey.Raw()
	if err != nil {
		return
	}
	als := &LedgerAccountLocalStorage{
		CryptoType: int32(account.PublicKey.Type()),
		PubKey:     hexutil.ToHex(pubKeyBytes),
		PrivKey:    hexutil.ToHex(privKeyBytes),
	}

	byteContent, err := json.MarshalIndent(als, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filePath, byteContent, 0600)
	return
}

type LedgerAccountGenerator struct {
	PrivateGenerator PrivateGenerator
	AddressConverter AddressConverter
}

func (g *LedgerAccountGenerator) Generate(typ types.CryptoType, src io.Reader) (account *types.OgLedgerAccount, err error) {
	privKey, pubKey, err := g.PrivateGenerator.GeneratePair(int(typ), src)
	if err != nil {
		return
	}

	addr, err := g.AddressConverter.AddressFromPubKey(pubKey)
	if err != nil {
		return
	}
	account = &types.OgLedgerAccount{
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
	byteContent, err := pubKey.Bytes()
	if err != nil {
		return
	}
	addr = &og_interface.Address20{}
	addr.FromBytes(Keccak256(byteContent))
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
	byteContent, err := pubKey.Bytes()
	if err != nil {
		return
	}
	w.Write(byteContent)
	hasher := ripemd160.New()
	hasher.Write(w.Bytes())
	result := hasher.Sum(nil)
	addr = &og_interface.Address20{}
	addr.FromBytes(result)
	return
}
