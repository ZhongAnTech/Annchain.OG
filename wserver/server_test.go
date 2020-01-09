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
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	addr := ":12345"
	srv := NewServer(addr)
	go func() {
		//time.Sleep(time.Second * time.Duration(5))
		for i := 0; i < 10000; i++ {
			srv.Push(EVENT_NEW_UNIT, fmt.Sprintf("msg %d", i))
			time.Sleep(time.Millisecond * time.Duration(500))
		}
	}()
	srv.Serve()

	time.Sleep(time.Second * 60)
}
