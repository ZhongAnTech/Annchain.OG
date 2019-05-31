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
package dkg

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/filename"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/annsensus/term"
	"github.com/sirupsen/logrus"
	"testing"

	"github.com/annchain/OG/types"
)

func TestDkg_VerifyBlsSig(t *testing.T) {
	seq := types.RandomSequencer()
	fmt.Println(seq.GetTxHash().Hex())
	seq.BlsJointPubKey = []byte{}
}

func logInit() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	filenameHook := filename.NewHook()
	filenameHook.Field = "line"
	logrus.AddHook(filenameHook)
	log = logrus.StandardLogger()
}
func TestVrfSelections_Le(t *testing.T) {
	logInit()

	d := NewDkg(true, 4, 3, nil, nil, nil, nil)
	tm := term.NewTerm(1, 21, 4)
	d.term = tm
	for i := 0; i < 21; i++ {
		h := types.RandomHash()
		cp := types.Campaign{
			Vrf: types.VrfInfo{
				Vrf: h.Bytes[:],
			},
			Issuer: types.RandomAddress(),
		}
		tm.AddCampaign(&cp)
	}
	seq := &types.Sequencer{
		BlsJointSig: types.RandomAddress().ToBytes(),
	}
	d.myAccount = getRandomAccount()
	d.SelectCandidates(seq)
	return
}

func getRandomAccount() *account.SampleAccount {
	_, priv := crypto.Signer.RandomKeyPair()
	return account.NewAccount(priv.String())
}

func TestDKgLog(t *testing.T) {
	logInit()
	d := NewDkg(true, 4, 3, nil, nil, nil, nil)
	log := log.WithField("me", d.GetId())
	log.Debug("hi")
	d.partner.Id = 10
	log.Info("hi hi")
}

func TestReset(t *testing.T) {
	logInit()
	d := NewDkg(true, 4, 3, nil, nil, nil, nil)
	log.Debug("sk ", d.partner.MyPartSec)
	d.Reset(nil)
	log.Debug("sk ", d.partner.MyPartSec)
	d.GenerateDkg()
	log.Debug("sk ", d.partner.MyPartSec)
	d.Reset(nil)
	log.Debug("sk ", d.partner.MyPartSec)

}

func TestNewDKGPartner(t *testing.T) {

	for j := 0; j < 22; j++ {
		fmt.Println(j, j*2/3+1)
	}
	for i := 0; i < 4; i++ {
		d := NewDkg(true, 4, 3, nil, nil, nil, nil)
		fmt.Println(hexutil.Encode(d.PublicKey()))
		fmt.Println(d.partner.MyPartSec)
	}

}
