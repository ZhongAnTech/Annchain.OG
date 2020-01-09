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
	"github.com/annchain/OG/types/tx_types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAnnSensus_VrfVerify(t *testing.T) {
	for i := 0; i < 100; i++ {
		a := &AnnSensus{}
		cp := &tx_types.Campaign{}
		a.Idag = &DummyDag{}
		vrf := a.GenerateVrf()
		fmt.Println(vrf, i)
		if vrf == nil {
			continue
		}
		cp.Vrf = *vrf
		err := a.VrfVerify(cp.Vrf.Vrf, cp.Vrf.PublicKey, cp.Vrf.Message, cp.Vrf.Proof)
		require.Nil(t, err)
		return
	}

}
