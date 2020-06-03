package transport

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
)

type TransportPrivateInfo struct {
	Type             string
	PrivateKeyString string
	PrivateKey       core.PrivKey
	Id               string
}

type DefaultTransportIdentityHolder struct {
	KeyFile string
	pi      *TransportPrivateInfo
}

func (a *DefaultTransportIdentityHolder) ProvidePrivateKey(createIfMissing bool) (identity *TransportPrivateInfo, err error) {
	if a.pi != nil {
		identity = a.pi
		return
	}

	err = a.loadPrivateKey(a.KeyFile)
	if err == nil {
		identity = a.pi
		return
	}
	if !createIfMissing {
		return // error
	}

	// create and reload
	logrus.Info("creating a new transport account")
	err = a.initPrivateInfo(nil)
	if err != nil {
		return
	}
	err = a.loadPrivateKey(a.KeyFile)
	if err == nil {
		identity = a.pi
	}

	return
}

func (a *DefaultTransportIdentityHolder) loadPrivateKey(keyFile string) error {
	// read key file
	bytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return err
	}

	a.pi = &TransportPrivateInfo{}
	err = json.Unmarshal(bytes, a.pi)
	if err != nil {
		return err
	}

	privb, err := hex.DecodeString(a.pi.PrivateKeyString)
	if err != nil {
		return err
	}

	pk, err := core.UnmarshalPrivateKey(privb)
	if err != nil {
		logrus.WithError(err).Fatal("failed to unmarshal private key")
	}
	a.pi.PrivateKey = pk
	return nil
}

func (a *DefaultTransportIdentityHolder) initPrivateInfo(randomizer io.Reader) error {
	if randomizer == nil {
		randomizer = crand.Reader
	}

	priv, _, err := core.GenerateKeyPairWithReader(core.Secp256k1, 0, randomizer)
	if err != nil {
		return err
	}

	privm, err := core.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}
	privs := hex.EncodeToString(privm)

	opts := []libp2p.Option{
		libp2p.Identity(priv),
	}

	ctx := context.Background()

	node, err := libp2p.New(ctx, opts...)
	if err != nil {
		return err
	}

	pi := &TransportPrivateInfo{
		Type:             "secp256k1",
		PrivateKeyString: privs,
		Id:               node.ID().String(),
	}

	bytes, err := json.MarshalIndent(pi, "", " ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(a.KeyFile, bytes, 0600)
	if err != nil {
		return err
	}
	return nil
}
