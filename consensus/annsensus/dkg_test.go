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
	"testing"

	"github.com/annchain/OG/types"
)

func TestDkg_VerifyBlsSig(t *testing.T) {
	seq := types.RandomSequencer()
	fmt.Println(seq.GetTxHash().Hex())
	seq.BlsJointPubKey = []byte{}
}

func TestVrfSelections_Le(t *testing.T) {
	logInit()
	ann := AnnSensus{
		Idag:           &DummyDag{},
		NbParticipants: 4,
	}
	d := newDkg(&ann, true, 4, 3)
	tm := newTerm(1, 21)
	ann.term = tm
	ann.dkg = d
	for i := 0; i < 21; i++ {
		h := types.RandomHash()
		cp := types.Campaign{
			Vrf: types.VrfInfo{
				Vrf: h.Bytes[:],
			},
			Issuer: types.RandomAddress(),
		}
		tm.campaigns[cp.Issuer] = &cp
	}
	d.SelectCandidates()
	return
}

func TestDKgLog(t *testing.T) {
	logInit()
	var d *Dkg
	d = newDkg(&AnnSensus{}, true, 4, 3)
	log := log.WithField("me", d.GetId())
	log.Debug("hi")
	d.partner.Id = 10
	log.Info("hi hi")
}

func TestReset(t *testing.T) {
	logInit()
	d := newDkg(&AnnSensus{}, true, 4, 3)
	log.Debug("sk ", d.partner.MyPartSec)
	d.Reset(nil)
	log.Debug("sk ", d.partner.MyPartSec)
	d.GenerateDkg()
	log.Debug("sk ", d.partner.MyPartSec)
	d.Reset(nil)
	log.Debug("sk ", d.partner.MyPartSec)

}
