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
package types

import (
	"fmt"
	"testing"
)

func TestNewSequencerHead(t *testing.T) {

	var seqs Sequencers
	for i := 0; i < 3; i++ {
		s := Sequencer{
			TxBase: TxBase{
				Hash:   randomHash(),
				Height: uint64(i),
			},
		}
		seqs = append(seqs, &s)
	}
	for i := 0; i < 3; i++ {
		fmt.Println(seqs[i])
		fmt.Println(*seqs[i])
	}

	headres := seqs.ToHeaders()
	for i := 0; i < 3; i++ {
		fmt.Println(headres[i])
		//fmt.Println(*headres[i])
	}
}
