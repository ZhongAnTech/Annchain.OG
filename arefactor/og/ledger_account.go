package og

import (
	"encoding/json"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"golang.org/x/crypto/sha3"
	"io"
	"io/ioutil"
)

type LedgerAccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
	Address    string
}

type LocalLedgerAccountHolder struct {
	PrivateGenerator og_interface.PrivateGenerator
	AddressConverter og_interface.AddressConverter
	BackFilePath     string
	CryptoType       og_interface.CryptoType
	account          *og_interface.OgLedgerAccount
}

func (l *LocalLedgerAccountHolder) ProvideAccount() (*og_interface.OgLedgerAccount, error) {
	if l.account == nil {
		return l.Load()
	}
	return l.account, nil
}

func (l *LocalLedgerAccountHolder) Account() *og_interface.OgLedgerAccount {
	return l.account
}

func (l *LocalLedgerAccountHolder) SetAccount(account *og_interface.OgLedgerAccount) {
	l.account = account
}

func (l *LocalLedgerAccountHolder) Load() (account *og_interface.OgLedgerAccount, err error) {
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

	account = &og_interface.OgLedgerAccount{
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

func (g *LocalLedgerAccountHolder) Generate(src io.Reader) (account *og_interface.OgLedgerAccount, err error) {
	privKey, pubKey, err := g.PrivateGenerator.GeneratePair(int(g.CryptoType), src)
	if err != nil {
		return
	}
	account = &og_interface.OgLedgerAccount{
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

type OgAddressConverter struct {
}

func (o *OgAddressConverter) AddressFromAccount(account *og_interface.OgLedgerAccount) (addr og_interface.Address, err error) {
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
