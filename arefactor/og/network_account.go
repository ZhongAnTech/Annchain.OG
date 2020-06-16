package og

import (
	"encoding/json"
	"github.com/annchain/OG/arefactor/common/hexutil"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
	"io/ioutil"
)

type TransportAccountHolder interface {
	ProvideAccount() (*transport_interface.TransportAccount, error)
	Generate(src io.Reader) (account *transport_interface.TransportAccount, err error)
	Load() (account *transport_interface.TransportAccount, err error)
	Save() (err error)
}

type LocalTransportAccountHolder struct {
	PrivateGenerator   PrivateGenerator
	NetworkIdConverter NetworkIdConverter
	BackFilePath       string
	CryptoType         transport_interface.CryptoType
	Account            *transport_interface.TransportAccount
}

func (l *LocalTransportAccountHolder) ProvideAccount() (*transport_interface.TransportAccount, error) {
	if l.Account == nil {
		return l.Load()
	}
	return l.Account, nil
}

type TransportAccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
	NetworkId  string
}

// only private key is mandatory.
func (l *LocalTransportAccountHolder) Load() (account *transport_interface.TransportAccount, err error) {
	bytes, err := ioutil.ReadFile(l.BackFilePath)
	if err != nil {
		return
	}
	als := &TransportAccountLocalStorage{}
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

	account = &transport_interface.TransportAccount{
		PublicKey:  privKey.GetPublic(),
		PrivateKey: privKey,
	}
	account.NodeId, err = l.NetworkIdConverter.NetworkIdFromAccount(account)
	l.Account = account
	return
}

func (l *LocalTransportAccountHolder) Save() (err error) {
	account := l.Account
	pubKeyBytes, err := account.PublicKey.Raw()
	if err != nil {
		return
	}
	privKeyBytes, err := account.PrivateKey.Raw()
	if err != nil {
		return
	}
	als := &TransportAccountLocalStorage{
		CryptoType: int32(account.PublicKey.Type()),
		PubKey:     hexutil.ToHex(pubKeyBytes),
		PrivKey:    hexutil.ToHex(privKeyBytes),
		NetworkId:  account.NodeId,
	}

	bytes, err := json.MarshalIndent(als, "", "    ")
	if err != nil {
		return
	}
	err = ioutil.WriteFile(l.BackFilePath, bytes, 0600)
	return
}

func (l *LocalTransportAccountHolder) Generate(src io.Reader) (account *transport_interface.TransportAccount, err error) {
	privKey, pubKey, err := l.PrivateGenerator.GeneratePair(int(l.CryptoType), src)
	if err != nil {
		return
	}
	account = &transport_interface.TransportAccount{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
	account.NodeId, err = l.NetworkIdConverter.NetworkIdFromAccount(account)
	return
}

type NetworkIdConverter interface {
	NetworkIdFromAccount(account *transport_interface.TransportAccount) (networkId string, err error)
}

// OgNetworkIdConverter converts private/public key to network id that libp2p use.
type OgNetworkIdConverter struct {
}

func (o *OgNetworkIdConverter) NetworkIdFromAccount(account *transport_interface.TransportAccount) (networkId string, err error) {
	id, err := peer.IDFromPublicKey(account.PublicKey)
	if err != nil {
		return
	}
	return id.String(), err
	//opts := []libp2p.Option{
	//	libp2p.Identity(account.PrivateKey),
	//}
	//
	//ctx := context.Background()
	//
	//node, err := libp2p.New(ctx, opts...)
	//if err != nil {
	//	return
	//}
	//networkId = node.ID().String()
	//return
}
