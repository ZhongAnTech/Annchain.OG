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
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3/pairing/bn256"
	"github.com/annchain/OG/common/hexutil"
	"testing"
)

func TestAnnSensus_GenerateDKgPublicKey(t *testing.T) {
	var as = NewAnnSensus(1, true, 5, 4)
	pk := as.dkg.pk
	fmt.Println(hexutil.Encode(pk))
	point, err := bn256.UnmarshalBinaryPointG2(pk)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(point)
}
