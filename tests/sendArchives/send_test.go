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
package sendArchives

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/common/httplib"
	"testing"
	"time"
)

type TxReq struct {
	Data string `json:"data"`
}

func TestArchives(t *testing.T) {
	Host := "http://192.168.45.149:11300"
	var i uint
	for {
		select {
		case <-time.After(time.Millisecond * 200):
			i++
			var data []byte
			if i > 255 {
				data = append(data, byte(i), byte(i), byte(i))
			} else {
				j := i / 255
				m := i % 255
				c := m / 255
				data = append(data, byte(c), byte(j), byte(m))
			}
			req := httplib.Post(Host + "/new_archive")

			txReq := TxReq{
				Data: string(data),
			}
			_, err := req.JSONBody(&txReq)
			if err != nil {
				panic(fmt.Errorf("encode tx errror %v", err))
			}
			d, _ := json.MarshalIndent(&txReq, "", "\t")
			fmt.Println(string(d))

			str, err := req.String()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(str)
		}
	}
}
