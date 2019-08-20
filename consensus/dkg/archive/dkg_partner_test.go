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
package archive

import (
	"fmt"
	"github.com/annchain/OG/account"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/OG/consensus/term"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestDkg_VerifyBlsSig(t *testing.T) {
	seq := tx_types.RandomSequencer()
	fmt.Println(seq.GetTxHash().Hex())
	seq.BlsJointPubKey = []byte{}
}

func logInit() {
	Formatter := new(logrus.TextFormatter)
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	//filenameHook := filename.NewHook()
	//filenameHook.Field = "line"
	//logrus.AddHook(filenameHook)
	dkg.log = logrus.StandardLogger()
}
func TestVrfSelections_Le(t *testing.T) {
	logInit()

	d := NewDkgPartner(true, 4, 3, nil, nil, nil)
	tm := term.NewTerm(1, 21, 4)
	d.term = tm
	for i := 0; i < 21; i++ {
		addr := common.RandomAddress()
		h := common.RandomHash()
		cp := tx_types.Campaign{
			Vrf: tx_types.VrfInfo{
				Vrf: h.Bytes[:],
			},
			Issuer: &addr,
		}
		tm.AddCampaign(&cp)
	}
	seq := &tx_types.Sequencer{
		BlsJointSig: common.RandomAddress().ToBytes(),
	}
	d.myAccount = getRandomAccount()
	d.SelectCandidates(seq)
	return
}

func getRandomAccount() *account.Account {
	_, priv := crypto.Signer.RandomKeyPair()
	return account.NewAccount(priv.String())
}

func TestDKgLog(t *testing.T) {
	logInit()
	d := NewDkgPartner(true, 4, 3, nil, nil, nil)
	log := dkg.log.WithField("me", d.GetId())
	log.Debug("hi")
	d.context.Id = 10
	log.Info("hi hi")
}

func TestReset(t *testing.T) {
	logInit()
	d := NewDkgPartner(true, 4, 3, nil, nil, nil)
	dkg.log.Debug("sk ", d.context.MyPartSec)
	d.Reset(nil)
	dkg.log.Debug("sk ", d.context.MyPartSec)
	d.GenerateDkg()
	dkg.log.Debug("sk ", d.context.MyPartSec)
	d.Reset(nil)
	dkg.log.Debug("sk ", d.context.MyPartSec)

}

func TestNewDKGPartner(t *testing.T) {

	for j := 0; j < 22; j++ {
		fmt.Println(j, j*2/3+1)
	}
	for i := 0; i < 4; i++ {
		d := NewDkgPartner(true, 4, 3, nil, nil, nil)
		fmt.Println(hexutil.Encode(d.PublicKey()))
		fmt.Println(d.context.MyPartSec)
	}

}
