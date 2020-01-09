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
package lock_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func testMain() {
	go func() {
		err := http.ListenAndServe("0.0.0.0:"+"1510", nil)
		if err != nil {
			panic(err)
		}
	}()
	var P = Person{
		Name: "hahah",
		ID:   345,
	}
	var g = Group{}
	g.SetHead(&P)
	fmt.Println("start")
	var q = make(chan bool)
	var num int
	go func() {
		for {
			select {
			case <-time.After(time.Second * 1):
				fmt.Println("hahahh", num)
				num++
				if num > 100 {
					q <- true
					return
				}
			}
		}
	}()
	for {
		select {
		case <-time.After(1 * time.Second):
			go fmt.Println(g.GetHead())
			go fmt.Println(g.GetHeadId())
			go fmt.Println(g.GetHead().ID)
		case <-q:
			fmt.Println("end")
			return

		}
	}
}

func TestLocker(t *testing.T) {
	testMain()

}

type Person struct {
	Name string
	ID   int
}

type Group struct {
	Head *Person
	mu   sync.RWMutex
	//
}

func (g *Group) SetHead(s *Person) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Head = s
}

func (g *Group) GetHead() (s *Person) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Head
}

func (g *Group) GetHeadId() (id int) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Head.ID
}
