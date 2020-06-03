package engine_test

import (
	"github.com/annchain/OG/arefactor/common/mylog"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/engine"
	"github.com/annchain/OG/message"
	"github.com/annchain/OG/plugin/annsensus"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func generatePeers(suite *bn256.Suite, n int) []dkg.PartSec {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	peerInfos := make([]dkg.PartSec, n)
	for i := 0; i < n; i++ {
		pubKey, privKey := signer.RandomKeyPair()
		address := pubKey.Address()
		// dkg kyber pub/priv key
		dkgPrivKey, dkgPubKey := dkg.GenPartnerPair(suite)

		peerInfos[i] = dkg.PartSec{
			PartPub: dkg.PartPub{
				Point: dkgPubKey,
				Peer: dkg.DkgPeer{
					Id:             i,
					PublicKey:      pubKey,
					Address:        address,
					PublicKeyBytes: nil,
				},
			},
			Scalar:     dkgPrivKey,
			PrivateKey: privKey,
		}
	}
	return peerInfos
}

// TestAnnsensusBenchmark will start the blockchain from genesis
func TestAnnsensusBenchmark(t *testing.T) {
	mylog.LogInit(logrus.InfoLevel)
	nodes := 4
	plugins := make([]*annsensus.AnnsensusPlugin, nodes)
	chans := make([]chan *message.GeneralMessageEvent, nodes)
	communicators := make([]*LocalGeneralPeerCommunicator, nodes)

	engines := make([]*engine.Engine, nodes)

	for i := 0; i < nodes; i++ {
		chans[i] = make(chan *message.GeneralMessageEvent)
	}

	for i := 0; i < nodes; i++ {
		communicators[i] = NewLocalGeneralPeerCommunicator(i, chans[i], chans)
	}

	accounts := sampleAccounts(nodes)

	for i := 0; i < nodes; i++ {
		plugins[i] = annsensus.NewAnnsensusPlugin(
			NewDummyTermProvider(),
			&dummyAccountProvider{MyAccount: accounts[i]},
			&dummyProposalGenerator{},
			&dummyProposalValidator{},
			&dummyDecisionMaker{})
		plugins[i].SetOutgoing(communicators[i])
	}

	// init general processor
	for i := 0; i < nodes; i++ {
		eng := engine.Engine{
			Config:       engine.EngineConfig{},
			PeerOutgoing: communicators[i],
			PeerIncoming: communicators[i],
		}
		eng.InitDefault()
		eng.RegisterPlugin(plugins[i])
		engines[i] = &eng
		eng.Start()
	}

	suite := bn256.NewSuiteG2()
	// genesis accounts for dkg. Only partpub is shared.
	// message is encrypted using partpub. TODO: is that possible? or we need an account public key
	peers := generatePeers(suite, nodes)
	pubs := make([]dkg.PartPub, len(peers))

	for i, peer := range peers {
		pubs[i] = peer.PartPub
	}
	// for consensus, all peers are participated in the consensus.
	senators := make([]term.Senator, len(accounts))
	// build a public shared term
	for i, account := range accounts {
		senators[i] = term.Senator{
			Id:           i,
			Address:      account.Address,
			PublicKey:    account.PublicKey,
			BlsPublicKey: pubs[i],
		}
	}

	for i := 0; i < nodes; i++ {
		c := plugins[i].AnnsensusPartner.TermProvider.GetTermChangeEventChannel()
		genesisTerm := &term.Term{
			Id:                0,
			PartsNum:          nodes,
			Threshold:         nodes*2/3 + 1,
			Senators:          senators,
			AllPartPublicKeys: pubs,
			PublicKey:         peers[i].PartPub.Point,
			ActivateHeight:    0,
			Suite:             suite,
		}

		//tm := term.NewTerm(1,nodes,0)
		contextProvider := dummyContext{
			term:      genesisTerm,
			MyBftId:   i,
			MyPartSec: peers[i],
		}

		c <- contextProvider
	}
	logrus.Info("Started")
	//plugins[0].OgPartner.SendMessagePing(communication.OgPeer{Id: 1})

	var lastValue uint = 0
	for i := 0; i < 60; i++ {
		v := engines[0].GetBenchmarks()["mps"].(uint)
		if lastValue == 0 {
			lastValue = v
		} else {
			logrus.WithField("mps", v-lastValue).Info("performance")
		}
		lastValue = v
		time.Sleep(time.Second)
	}
}
