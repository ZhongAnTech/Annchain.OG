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
package testChan

import (
	"fmt"
	"testing"
)

type s struct {
	ch      chan bool
	stopped bool
}

func TestA(t *testing.T) {

	var a = make(chan bool)
	var pa = &a
	var b chan bool
	var pb = &b
	fmt.Println(a, b, pa, pb)
	b = a
	pb = pa
	fmt.Println(a, b, pa, pb)
	a = nil
	pa = nil
	fmt.Println(a, b, pa, pb)

}

func TestB(t *testing.T) {
	var a = s{
		ch:      make(chan bool),
		stopped: false,
	}
	var c = &a
	var b = a
	fmt.Println(a, c, b)
	a.stopped = true
	fmt.Println(a, c, b)
}
