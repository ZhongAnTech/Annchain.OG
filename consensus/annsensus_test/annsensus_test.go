// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package annsensus_test

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

//func genPublicKeys(num int) (accounts []crypto.PublicKey) {
//	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
//	for i := 0; i < num; i++ {
//		pub, _ := signer.RandomKeyPair()
//		accounts = append(accounts, pub)
//	}
//	return accounts
//}
//
//func TestAnnSensus_GenerateDKgPublicKey(t *testing.T) {
//	var as = NewAnnSensus(4, false, 1, true, 5,
//		genPublicKeys(5), "test.json", false)
//
//	pk := as.dkg.PublicKey()
//	fmt.Println(hexutil.Encode(pk))
//	point, err := bn256.UnmarshalBinaryPointG2(pk)
//	if err != nil {
//		t.Fatal(err)
//	}
//	fmt.Println(point)
//}

func logInit() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	Formatter.ForceColors = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	//logrus.SetReportCaller(true)
}

func init() {
	logInit()
}

func generatePeers(suite *bn256.Suite, n int) []dkg.PartSec {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	var peerInfos []dkg.PartSec
	for i := 0; i < n; i++ {
		pubKey, privKey := signer.RandomKeyPair()
		address := pubKey.Address()
		// dkg kyber pub/priv key
		dkgPrivKey, dkgPubKey := dkg.GenPartnerPair(suite)

		peerInfos = append(peerInfos, dkg.PartSec{
			PartPub: dkg.PartPub{
				Point: dkgPubKey,
				Peer: dkg.PeerInfo{
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

func TestAnnSensusFourNodesGenesisTerm(t *testing.T) {
	nodes := 4
	accounts := sampleAccounts(nodes)

	suite := bn256.NewSuiteG2()

	// build an annsensus
	config := annsensus.AnnsensusProcessorConfig{
		DisableTermChange:  false,
		DisabledConsensus:  false,
		TermChangeInterval: 60 * 1000,
		GenesisAccounts:    nil,
		PartnerNum:         nodes,
	}

	// prepare message channel for each peer
	// both bft and dkg messages will be adapted to AnnsensusMessage
	peerChans := make([]chan annsensus.AnnsensusMessage, nodes)
	for i := 0; i < nodes; i++ {
		peerChans[i] = make(chan annsensus.AnnsensusMessage, 50)
	}

	aps := make([]*annsensus.AnnsensusProcessor, nodes)
	termProviders := make([]annsensus.TermIdProvider, nodes)

	for i := 0; i < nodes; i++ {
		// init AnnsensusPeerCommunicator for each node
		bftAdapter := &annsensus.PlainBftAdapter{}
		dkgAdapter := &annsensus.PlainDkgAdapter{}
		communicator := &LocalAnnsensusPeerCommunicator{
			Myid:  i,
			Peers: peerChans,
			pipe:  peerChans[i],
		}

		termProvider := NewDummyTermProvider()
		termHolder := annsensus.NewAnnsensusTermHolder(termProvider)
		defaultAnnsensusPartnerProvider := annsensus.NewDefaultAnnsensusPartnerProvider(
			&dummyAccountProvider{MyAccount: accounts[i]},
			&dummyProposalGenerator{},
			&dummyProposalValidator{},
			&dummyDecisionMaker{},
			bftAdapter,
			dkgAdapter,
			communicator,
		)

		ann := annsensus.NewAnnsensusProcessor(config,
			bftAdapter, dkgAdapter, communicator,
			termProvider, termHolder,
			defaultAnnsensusPartnerProvider,
			defaultAnnsensusPartnerProvider,
		)
		aps = append(aps, ann)
		termProviders = append(termProviders, termProvider)
	}

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

	genesisTerm := &term.Term{
		Id:                0,
		PartsNum:          len(peers),
		Threshold:         len(peers)*2/3 + 1,
		Senators:          senators,
		AllPartPublicKeys: pubs,
		PublicKey:         nil,
		ActivateHeight:    0,
		Suite:             nil,
	}
	// recover the public key now since everyone (including light peers) needs this info to verify txs.

	for i := 0; i < nodes; i++ {
		aps[i].Start()
		c := termProviders[i].GetTermChangeEventChannel()

		//tm := term.NewTerm(1,nodes,0)
		contextProvider := dummyContextProvider{
			term:      genesisTerm,
			MyBftId:   i,
			MyPartSec: dkg.PartSec{},
		}

		c <- contextProvider
	}

	time.Sleep(time.Hour * 1)
}
