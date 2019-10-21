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
package txcache

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/og/protocol/ogmessage"

	"github.com/annchain/gcache"
	"github.com/sirupsen/logrus"
	"sync"
	"testing"
	"time"
)

func init_log() {
	Formatter := new(logrus.TextFormatter)
	//Formatter.DisableColors = true
	Formatter.TimestampFormat = "2006-01-02 15:04:05.000000"
	Formatter.FullTimestamp = true

	logrus.SetFormatter(Formatter)
}

func (c *TxCache) addInitTx() {
	c.addInitTxWithNum(40000)
}

func (c *TxCache) addInitTxWithNum(num int) {
	minWeight := uint64(500)
	for i := 0; i < num; i++ {
		var tx ogmessage.Txi
		if i%40 == 0 {
			tx = ogmessage.RandomSequencer()
			tx.GetBase().Height = uint64(i / 1000)
			tx.GetBase().Weight = uint64(i%1000) + tx.GetBase().Height + minWeight
		} else {
			tx = ogmessage.RandomTx()
			tx.GetBase().Height = uint64(i / 1000)
			tx.GetBase().Weight = uint64(i%1000) + tx.GetBase().Height + minWeight
		}
		err := c.EnQueue(tx)
		if err != nil {
			panic(err)
		}
		//fmt.Println(tx)
	}
}

func newTestTxcache(sorted bool) *TxCache {
	return newTestTxcacheWithParam(100000, sorted, 3)
}

func newTestTxcacheWithParam(size int, sorted bool, expire int) *TxCache {
	logrus.SetLevel(logrus.InfoLevel)
	invalidTx := func(h common.Hash) bool {
		return true
	}
	//gcache.DebugMode = true
	c := NewTxCache(size, expire, invalidTx, sorted)

	logrus.SetLevel(logrus.DebugLevel)
	return c
}

func TestTxCache_Get(t *testing.T) {
	start := time.Now()
	tx1 := ogmessage.RandomTx()
	tx2 := ogmessage.RandomTx()
	c := newTestTxcache(true)
	c.addInitTx()
	c.EnQueue(tx1)
	if !c.Has(tx1.GetTxHash()) {
		t.Fatal("tx is in cache")
	}
	if c.Has(tx2.GetTxHash()) {
		t.Fatal("tx is not in cache")
	}
	hash := c.GetHashOrder()[5]
	t.Log("hash", hash)
	if tx := c.Get(hash); tx != nil {
		t.Log(tx.String())
	} else {
		t.Fatal("tx not found")
	}
	if tx := c.Get(tx1.GetTxHash()); tx != tx1 {
		t.Fatalf("requied %v ;got %v", tx1.String(), tx.String())
	}
	t.Log("len", c.Len())
	t.Log(time.Now().Sub(start).String())
	c.Remove(hash)
	t.Log("remove tx", time.Now().Sub(start).String())
	time.Sleep(time.Second * 4)
	c.Refresh()
	if c.cache.Len() != 0 {
		t.Fatalf("requied 0 , got %d", c.Len())
	}
	if tx := c.Get(tx1.GetTxHash()); tx != nil {
		t.Fatalf("requied nil  ;got %v", tx.String())
	}
	c.Remove(tx1.GetTxHash())
	t.Log("len", c.Len())
	c.RemoveExpiredAndInvalid(100)
	t.Log("len", c.Len())
}

func TestTxCache_PopALl(t *testing.T) {

	c := newTestTxcache(true)
	c.addInitTx()
	fmt.Println(c.Len())
	start := time.Now()
	for c.Len() != 0 {
		c.DeQueue() //10000 item 2.1s
		//c.cache.Remove(hash) // 10000 item 6ms
	}
	if c.Len() != 0 {
		t.Fatalf("requied 0 , got %d", c.Len())
	}
	t.Log("deQueue all used", time.Now().Sub(start).String())
	c = newTestTxcache(true)
	c.addInitTx()
	start = time.Now()
	hashes := c.GetHashOrder()
	for _, hash := range hashes {
		//c.deQueue()     //10000 item 2.1s
		c.cache.Remove(hash) // 10000 item 6ms
	}
	if c.Len() != 0 {
		t.Fatalf("requied 0 , got %d", c.Len())
	}
	t.Log("directly remove  all used", time.Now().Sub(start).String())

	c = newTestTxcache(true)
	c.addInitTx()
	start = time.Now()
	c.cache.DeQueueBatch(39000)
	c.cache.DeQueueBatch(3000)
	if c.Len() != 0 {
		t.Fatalf("requied 0 , got %d", c.Len())
	}
	t.Log("deque batch remove  all used", time.Now().Sub(start).String())
	//result for 50000 items
	//remove all by hash cmp used 28.865s
	//deQueue all used 22ms
	//directly remove  all used 22ms
}

func TestTxCache_EnQueue(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 28000; i++ {
		tx := ogmessage.RandomTx()
		wg.Add(1)
		go func() {
			//begin := time.Now()
			c.EnQueue(tx)
			//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
			wg.Done()
			//fmt.Println(time.Now().Sub(begin).String(),"done")
		}()
	}
	wg.Wait()
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
}

func TestTxCache_GetTop(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < 28000; i++ {
		tx := ogmessage.RandomTx()
		wg.Add(1)
		go func(int) {
			//begin := time.Now()
			c.EnQueue(tx)
			//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
			if i%1000 == 0 {
				fmt.Println("top,i", c.GetTop())
			} else if i%1001 == 0 {
				fmt.Println(c.DeQueueBatch(2))
			}
			wg.Done()
			//fmt.Println(time.Now().Sub(begin).String(),"done")
		}(i)
	}
	wg.Wait()
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
}

func TestTxCache_DeQueueBatch(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	c.DeQueueBatch(50000)
	start := time.Now()
	var txis []*ogmessage.Tx
	for i := 0; i < 100; i++ {
		tx := ogmessage.RandomTx()
		tx.Weight = uint64(i)
		txis = append(txis, tx)
		//begin := time.Now()
		c.EnQueue(tx)
		//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
		if i%20 == 0 {
			fmt.Println("top,i", c.GetTop())
		}
		//fmt.Println(time.Now().Sub(begin).String(),"done")
	}

	for i := 0; i < 100; i++ {
		tx := c.DeQueue()
		if tx != txis[i] {
			t.Fatal("diff ", tx, txis[i])
		}
		if i%20 == 0 {
			t.Log("same ", tx, txis[i])
		}
	}
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
}

func TestTxCache_AddFrontBatch(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	var txs []ogmessage.Txi
	for i := 0; i < 100; i++ {
		tx := ogmessage.RandomTx()
		tx.Weight = uint64(i*3 + 2)
		txs = append(txs, tx)
		//begin := time.Now()
		//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
		if i%20 == 0 {
			fmt.Println("top,i", c.GetTop())
		}
		//fmt.Println(time.Now().Sub(begin).String(),"done")
	}
	start := time.Now()
	c.PrependBatch(txs)
	for i := 0; i < 100; i++ {
		tx := c.DeQueue()
		if tx != txs[i] {
			t.Fatal("diff ", tx, txs[i])
		}
		if i%20 == 0 {
			t.Log("same ", tx, txs[i])
		}
	}
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
}

func TestTxCache_Sort(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	for i := 0; i < 100; i++ {
		fmt.Println(i, c.DeQueue())
	}
	//c.cache.PrintValues(1)
	fmt.Println("\n***sort**")
	start := time.Now()
	c.Sort()
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
	c.cache.PrintValues(1)
	for i := 0; i < 100; i++ {
		fmt.Println(i, c.DeQueue())
	}
}

func TestTxCache_Sort2(t *testing.T) {
	c := newTestTxcache(true)
	c.addInitTx()
	c.DeQueueBatch(100000)
	for i := 10; i < 40000; i++ {
		var tx ogmessage.Txi
		if i%40 == 0 {
			seq := ogmessage.RandomSequencer()
			if i%200 == 0 {
				seq.Height = uint64(i/1000 + 1)
				seq.Weight = uint64(i%1000) + seq.Height
			} else {
				seq.Height = uint64(i/1000 - 1)
				seq.Weight = uint64(i%1000) + seq.Height
			}

			tx = seq
		} else {
			randTx := ogmessage.RandomTx()
			randTx.Height = uint64(i / 100)
			randTx.Weight = uint64(i%100) + randTx.Height
			tx = randTx
		}
		err := c.EnQueue(tx)
		if err != nil {
			panic(err)
		}
	}
	for i := 0; i < 100; i++ {
		fmt.Println(i, c.DeQueue())
	}
	fmt.Println("\n***sort**")
	start := time.Now()
	c.Sort()
	fmt.Println(c.cache.Len(), "used", time.Now().Sub(start))
	for i := 0; i < 100; i++ {
		fmt.Println(i, c.DeQueue())
	}
}

func TestTxCache_Remove(t *testing.T) {
	c := newTestTxcacheWithParam(400000, true, 60)
	c.addInitTxWithNum(100000)
	gcache.DebugMode = true
	hash := c.GetHashOrder()[99999]
	tx := c.Get(hash)
	start := time.Now()
	c.Remove(hash)
	//gcache.DebugMode= false
	fmt.Println("sorted remove use time ", time.Now().Sub(start), hash, tx, c.cache.Len(), c.Get(hash))
	c = newTestTxcacheWithParam(400000, false, 60)
	c.addInitTxWithNum(100000)
	gcache.DebugMode = true
	hash = c.GetHashOrder()[99999]
	tx = c.Get(hash)
	start = time.Now()
	c.Remove(hash)
	fmt.Println("unsorted  remove use time ", time.Now().Sub(start), hash, tx, c.cache.Len(), c.Get(hash))
}

func TestTxCache_Has(t *testing.T) {
	c := newTestTxcacheWithParam(1000, true, 60)
	c.addInitTxWithNum(200)
	gcache.DebugMode = true
	hash := c.GetHashOrder()[150]
	tx := c.Get(hash)
	fmt.Println(hash, tx, c.cache.Len(), c.Get(hash), c.Has(hash), c.Has(ogmessage.RandomTx().GetTxHash()))
}
