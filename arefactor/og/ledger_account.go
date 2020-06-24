package og

import (
	"encoding/json"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"golang.org/x/crypto/sha3"
	"io"
	"io/ioutil"
)

type LedgerAccountHolder interface {
	ProvideAccount() (*types.OgLedgerAccount, error)
	Generate(src io.Reader) (account *types.OgLedgerAccount, err error)
	Load() (account *types.OgLedgerAccount, err error)
	Save() (err error)
}

type LedgerAccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
	Address    string
}

type LocalLedgerAccountHolder struct {
	PrivateGenerator PrivateGenerator
	AddressConverter AddressConverter
	BackFilePath     string
	CryptoType       types.CryptoType
	account          *types.OgLedgerAccount
}

func (l *LocalLedgerAccountHolder) ProvideAccount() (*types.OgLedgerAccount, error) {
	if l.account == nil {
		return l.Load()
	}
	return l.account, nil
}

func (l *LocalLedgerAccountHolder) Account() *types.OgLedgerAccount {
	return l.account
}

func (l *LocalLedgerAccountHolder) SetAccount(account *types.OgLedgerAccount) {
	l.account = account
}

func (l *LocalLedgerAccountHolder) Load() (account *types.OgLedgerAccount, err error) {
	byteContent, err := ioutil.ReadFile(l.BackFilePath)
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
	account.Address, err = l.AddressConverter.AddressFromAccount(account)
	l.account = account
	return
}

func (l *LocalLedgerAccountHolder) Save() (err error) {
	account := l.account
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
		Address:    account.Address.AddressString(),
	}

	byteContent, err := json.MarshalIndent(als, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(l.BackFilePath, byteContent, 0600)
	return
}

func (g *LocalLedgerAccountHolder) Generate(src io.Reader) (account *types.OgLedgerAccount, err error) {
	privKey, pubKey, err := g.PrivateGenerator.GeneratePair(int(g.CryptoType), src)
	if err != nil {
		return
	}
	account = &types.OgLedgerAccount{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
	addr, err := g.AddressConverter.AddressFromAccount(account)
	if err != nil {
		return
	}
	account.Address = addr

	return
}

type AddressConverter interface {
	AddressFromAccount(account *types.OgLedgerAccount) (addr og_interface.Address, err error)
}

type OgAddressConverter struct {
}

func (o *OgAddressConverter) AddressFromAccount(account *types.OgLedgerAccount) (addr og_interface.Address, err error) {
	byteContent, err := account.PublicKey.Bytes()
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

//func (o *OgAddressConverter) AddressFromPubKey1(pubKey crypto.PubKey) (addr og_interface.Address, err error) {
//	var w bytes.Buffer
//	byteContent, err := pubKey.Bytes()
//	if err != nil {
//		return
//	}
//	w.Write(byteContent)
//	hasher := ripemd160.New()
//	hasher.Write(w.Bytes())
//	result := hasher.Sum(nil)
//	addr = &og_interface.Address20{}
//	addr.FromBytes(result)
//	return
//}
