package txcache

import (
	"fmt"
	"github.com/annchain/OG/types"
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

func newTestTxcache()*TxCache {
	logrus.SetLevel(logrus.InfoLevel)
	invalidTx := func (h types.Hash) bool {
		return true
	}
	c := NewTxCache(100000,3,invalidTx)
	for i:=0; i<40000;i++ {
		var tx types.Txi
		if i%40==0 {
			tx = types.RandomSequencer()
		}else {
			tx = types.RandomTx()
		}
		err := c.EnQueue(tx)
		if err!=nil {
			panic(err)
		}
	}
	logrus.SetLevel(logrus.DebugLevel)
	return c
}

func TestTxCache_Get(t *testing.T) {
	 start := time.Now()
	 tx1 := types.RandomTx()
	 tx2 := types.RandomTx()
	 c := newTestTxcache()
	 c.EnQueue(tx1)
	 if !c.Has(tx1.GetTxHash()) {
	 	t.Fatal("tx is in cache")
	 }
	 if c.Has(tx2.GetTxHash()) {
	 	t.Fatal("tx is not in cache")
	 }
	 hash := c.GetHashOrder()[5]
	 t.Log("hash",hash)
	 if tx := c.Get(hash) ;tx!=nil {
	 	t.Log(tx.String())
	 }else {
	 	t.Fatal("tx not found")
	 }
	if tx:= c.Get(tx1.GetTxHash());tx!=tx1 {
		t.Fatalf("requied %v ;got %v",tx1.String(),tx.String())
	}
	t.Log("len",c.Len())
	t.Log(time.Now().Sub(start).String())
	c.Remove(hash)
	t.Log("remove tx",time.Now().Sub(start).String())
	 time.Sleep(time.Second*4)
	c.Refresh()
	if c.cache.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	 if tx:= c.Get(tx1.GetTxHash());tx!=nil {
	 	t.Fatalf("requied nil  ;got %v",tx.String())
	 }
	 c.Remove(tx1.GetTxHash())
	 t.Log("len",c.Len())
	c.RemoveExpiredAndInvalid(100)
	t.Log("len",c.Len())
}

func TestTxCache_PopALl(t *testing.T) {

	c := newTestTxcache()
	fmt.Println(c.Len())
	start  := time.Now()
	for c.Len() !=0  {
		c.DeQueue()     //10000 item 2.1s
		//c.cache.Remove(hash) // 10000 item 6ms
	}
	if c.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	t.Log("deQueue all used", time.Now().Sub(start).String())
	c = newTestTxcache()
	start  = time.Now()
	hashes := c.GetHashOrder()
	for _, hash := range hashes {
		//c.deQueue()     //10000 item 2.1s
		c.cache.Remove(hash) // 10000 item 6ms
	}
	if c.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	t.Log("directly remove  all used", time.Now().Sub(start).String())


	c = newTestTxcache()
	start  = time.Now()
     c.cache.DeQueueBatch(39000)
	c.cache.DeQueueBatch(3000)
	if c.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	t.Log("dequebatch remove  all used", time.Now().Sub(start).String())
   //result for 50000 items
   //remove all by hash cmp used 28.865s
   //deQueue all used 22ms
   //directly remove  all used 22ms
}

func TestTxCache_EnQueue(t *testing.T) {
	c:= newTestTxcache()
	var wg sync.WaitGroup
	start := time.Now()
	for i:=0; i<28000;i++{
		tx:= types.RandomTx()
		wg.Add(1)
		go func (){
			//begin := time.Now()
			c.EnQueue(tx)
			//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
			wg.Done()
			//fmt.Println(time.Now().Sub(begin).String(),"done")
		}()
	}
	wg.Wait()
	fmt.Println(c.cache.Len(),"used",time.Now().Sub(start))
}


func TestTxCache_GetTop(t *testing.T) {
	c:= newTestTxcache()
	var wg sync.WaitGroup
	start := time.Now()
	for i:=0; i<28000;i++{
		tx:= types.RandomTx()
		wg.Add(1)
		go func (int ){
			//begin := time.Now()
			c.EnQueue(tx)
			//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
			if i%1000==0 {
				fmt.Println("top,i", c.GetTop())
			}else if i%1001 ==0 {
				fmt.Println(c.DeQueueBatch(2))
			}
			wg.Done()
			//fmt.Println(time.Now().Sub(begin).String(),"done")
		}(i)
	}
	wg.Wait()
	fmt.Println(c.cache.Len(),"used",time.Now().Sub(start))
}


func TestTxCache_DeQueueBatch(t *testing.T) {
	c:= newTestTxcache()
	c.DeQueueBatch(50000)
	start := time.Now()
	var txis [] *types.Tx
	for i:=0; i<100;i++{
		tx:= types.RandomTx()
		txis = append(txis,tx)
			//begin := time.Now()
			c.EnQueue(tx)
			//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
			if i%20==0 {
				fmt.Println("top,i", c.GetTop())
			}
			//fmt.Println(time.Now().Sub(begin).String(),"done")
		}

	for i:=0; i<100;i++{
		tx := c.DeQueue()
		if tx!= txis[i] {
			t.Fatal("diff ", tx ,txis[i])
		}
		if i%20==0 {
			t.Log("same ", tx ,txis[i])
		}
	}
	fmt.Println(c.cache.Len(),"used",time.Now().Sub(start))
}

func TestTxCache_AddFrontBatch(t *testing.T) {
	c:= newTestTxcache()
	start := time.Now()
	var txis [] *types.Tx
	for i:=0; i<100;i++{
		tx:= types.RandomTx()
		txis = append(txis,tx)
		//begin := time.Now()
		//fmt.Println(time.Now().Sub(begin).String(),"enqueue")
		if i%20==0 {
			fmt.Println("top,i", c.GetTop())
		}
		//fmt.Println(time.Now().Sub(begin).String(),"done")
	}
    c.AddFrontBatchTxs(txis)
	for i:=0; i<100;i++{
		tx := c.DeQueue()
		if tx!= txis[i] {
			t.Fatal("diff ", tx ,txis[i])
		}
		if i%20==0 {
			t.Log("same ", tx ,txis[i])
		}
	}
	fmt.Println(c.cache.Len(),"used",time.Now().Sub(start))
}