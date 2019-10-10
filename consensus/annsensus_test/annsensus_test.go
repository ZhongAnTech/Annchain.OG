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
	"github.com/annchain/OG/consensus/annsensus"
	"github.com/annchain/OG/consensus/bft"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"testing"
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
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
}

func init() {
	logInit()
}

func TestAnnSensusTwoNodes(t *testing.T) {
	nodes := 2
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

	// prepare bft channels
	var peerChansBft []chan *bft.BftMessage
	var peerChansDkg []chan *dkg.DkgMessage
	for i := 0; i < nodes; i++ {
		peerChansBft = append(peerChansBft, make(chan *bft.BftMessage, 5))
		peerChansDkg = append(peerChansDkg, make(chan *dkg.DkgMessage, 5))
	}

	var aps []*annsensus.AnnsensusProcessor
	annsensusPartnerProvider := NewDummyAnnsensusPartnerProivder(peerChansBft, peerChansDkg)


	for i := 0; i < nodes; i++ {
		commuicatorBft := NewDummyBftPeerCommunicator(i, peerChansBft[i], peerChansBft)
		commuicatorDkg := NewDummyDkgPeerCommunicator(i, peerChansDkg[i], peerChansDkg)

		ann := annsensus.NewAnnsensusProcessor(config,
			&dummySignatureProvider{},
			//&dummyContextProvider{
			//	NbParticipants: nodes,
			//	NbParts:        nodes,
			//	Threshold:      nodes,
			//	MyBftId:        i,
			//	BlockTime:      time.Second * 100,
			//	Suite:          suite,
			//	AllPartPubs:    nil,
			//	MyPartSec:      dkg.PartSec{},
			//},
			&dummyTermProvider{},
		)
		aps = append(aps, ann)
		// start two communicators
		go commuicatorBft.Run()
		go commuicatorDkg.Run()
	}
	for i := 0; i < nodes; i++ {
		aps[i].Start()
		// TODO: how to start?

	}
}
