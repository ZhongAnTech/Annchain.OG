package og

import (
	"encoding/json"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/transport_interface"
	"github.com/annchain/commongo/hexutil"
	"github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"io/ioutil"
)

type TransportAccountProvider interface {
	ProvideAccount() (*transport_interface.TransportAccount, error)
	Generate() (account *transport_interface.TransportAccount, err error)
	Load() (account *transport_interface.TransportAccount, err error)
	Save() (err error)
}

type TransportAccountLocalStorage struct {
	CryptoType int32
	PubKey     string
	PrivKey    string
	NetworkId  string
}

type LocalTransportAccountProvider struct {
	PrivateGenerator   og_interface.PrivateGenerator
	NetworkIdConverter NetworkIdConverter
	BackFilePath       string
	CryptoType         transport_interface.CryptoType
	account            *transport_interface.TransportAccount
}

func (l *LocalTransportAccountProvider) ProvideAccount() (*transport_interface.TransportAccount, error) {
	if l.account == nil {
		return l.Load()
	}
	return l.account, nil
}

func (l *LocalTransportAccountProvider) Account() *transport_interface.TransportAccount {
	return l.account
}

func (l *LocalTransportAccountProvider) SetAccount(account *transport_interface.TransportAccount) {
	l.account = account
}

// only private key is mandatory.
func (l *LocalTransportAccountProvider) Load() (account *transport_interface.TransportAccount, err error) {
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
	l.account = account
	return
}

func (l *LocalTransportAccountProvider) Save() (err error) {
	account := l.account
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

func (l *LocalTransportAccountProvider) Generate() (account *transport_interface.TransportAccount, err error) {
	privKey, pubKey, err := l.PrivateGenerator.GeneratePair(int(l.CryptoType))
	if err != nil {
		return
	}
	account = &transport_interface.TransportAccount{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}

	nodeId, err := l.NetworkIdConverter.NetworkIdFromAccount(account)
	if err != nil {
		return
	}
	account.NodeId = nodeId
	l.account = account
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
