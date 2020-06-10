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
package ogcrypto

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/arefactor/common/math"
	"testing"
)

func TestFromECDSA(t *testing.T) {
	priv, err := GenerateKey()
	fmt.Println(err, priv)
	data := math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
	fmt.Println(hex.EncodeToString(data))
}
