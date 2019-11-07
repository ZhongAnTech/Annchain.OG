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
	"github.com/annchain/OG/common/crypto"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestNewTerm(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	term := NewTerm(1, 3, 4)
	pk, _ := crypto.Signer.RandomKeyPair()
	term.PublicKeys = append(term.PublicKeys, pk)
	fmt.Println()
	term.ChangeTerm(&tx_types.TermChange{}, 2)
	fmt.Println(term.GetFormerPks())
}
