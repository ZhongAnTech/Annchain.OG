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
package annsensus

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/sirupsen/logrus"
	"testing"
)

func genPublicKeys(num int) (accounts []crypto.PublicKey) {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	for i := 0; i < num; i++ {
		pub, _ := signer.RandomKeyPair()
		accounts = append(accounts, pub)
	}
	return accounts
}

func TestAnnSensus_GenerateDKgPublicKey(t *testing.T) {
	var as = NewAnnSensus(4, false, 1, true, 5,
		genPublicKeys(5), "test.json", false)

	pk := as.dkg.PublicKey()
	fmt.Println(hexutil.Encode(pk))
	point, err := bn256.UnmarshalBinaryPointG2(pk)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(point)
}

func logInit() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
	log = logrus.StandardLogger()
}

func init() {
	logInit()
}
