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
package wserver

import (
	"fmt"
	"github.com/annchain/OG/types"
	"testing"
)

func TestConvertor(t *testing.T) {
	tx := types.Tx{
		TxBase: types.TxBase{
			Hash:        types.BytesToHash([]byte{1, 2, 3, 4, 5}),
			ParentsHash: types.Hashes{types.BytesToHash([]byte{1, 1, 2, 2, 3, 3})},
		},
		From:  types.HexToAddress("0x12345"),
		To:    types.HexToAddress("0x56789"),
		Value: nil,
	}
	fmt.Println(tx2UIData(tx))
}
