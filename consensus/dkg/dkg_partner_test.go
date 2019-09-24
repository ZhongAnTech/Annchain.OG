package dkg

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

var TestNodes = 4

func init() {
	Formatter := new(logrus.TextFormatter)
	//Formatter.ForceColors = false
	Formatter.DisableColors = true
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(Formatter)
	//logrus.SetReportCaller(true)

	//filenameHook := filename.NewHook()
	//filenameHook.Field = "line"
	//logrus.AddHook(filenameHook)
}

func generatePeers(suite *bn256.Suite, n int) []PartSec {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	var peerInfos []PartSec
	for i := 0; i < n; i++ {
		pubKey, privKey := signer.RandomKeyPair()
		address := pubKey.Address()
		// dkg kyber pub/priv key
		dkgPrivKey, dkgPubKey := GenPartnerPair(suite)

		peerInfos = append(peerInfos, PartSec{
			PartPub: PartPub{
				Point: dkgPubKey,
				Peer: PeerInfo{
					Id:             i,
					PublicKey:      pubKey,
					Address:        address,
					PublicKeyBytes: nil,
				},
			},
			Scalar:     dkgPrivKey,
			PrivateKey: privKey,
		})
	}
	return peerInfos
}

func setupPartners(termId uint32, numParts int, threshold int) ([]*DkgPartner, []PartSec) {
	suite := bn256.NewSuiteG2()

	// generate PeerInfos
	PartSecs := generatePeers(suite, numParts)
	var partPubs []PartPub
	for _, peer := range PartSecs {
		partPubs = append(partPubs, peer.PartPub)
	}

	var peerChans []chan DkgMessage

	// prepare incoming channels
	for i := 0; i < numParts; i++ {
		peerChans = append(peerChans, make(chan DkgMessage, 5000))
	}

	var partners []*DkgPartner

	for i := 0; i < numParts; i++ {
		partner, err := NewDkgPartner(suite, termId, numParts, threshold, partPubs, PartSecs[i])
		if err != nil {
			panic(err)
		}
		communicator := NewDummyDkgPeerCommunicator(i, peerChans[i], peerChans)
		partner.PeerCommunicator = communicator
		communicator.Run()

		partners = append(partners, partner)
	}
	return partners, PartSecs
}

type dummyDkgGeneratedListener struct {
	Wg       *sync.WaitGroup
	c        chan bool
	finished bool
}

func NewDummyDkgGeneratedListener(wg *sync.WaitGroup) *dummyDkgGeneratedListener {
	d := &dummyDkgGeneratedListener{
		Wg: wg,
		c:  make(chan bool),
	}
	go func() {
		for {
			<-d.c
			d.Wg.Done()
			logrus.Info("Dkg is generated")
			//break
		}
	}()
	return d
}

func (d dummyDkgGeneratedListener) GetDkgGeneratedEventChannel() chan bool {
	return d.c
}

func TestDkgPartner(t *testing.T) {
	// simulate 4 dkg partners
	termId := uint32(0)
	numParts := TestNodes
	threshold := TestNodes

	partners, _ := setupPartners(termId, numParts, threshold)
	wg := sync.WaitGroup{}

	wg.Add(len(partners))
	listener := NewDummyDkgGeneratedListener(&wg)

	for _, partner := range partners {
		partner.DkgGeneratedListeners = append(partner.DkgGeneratedListeners, listener)
		partner.Start()
	}
	time.Sleep(time.Second * 5)
	for _, partner := range partners {
		partner.gossipStartCh <- true
	}

	wg.Wait()
}
