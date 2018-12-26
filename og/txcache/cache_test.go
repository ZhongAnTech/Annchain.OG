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
	c := NewTxCache(100000,3)
	for i:=0; i<40000;i++ {
		var tx types.Txi
		if i%40==0 {
			tx = types.RandomSequencer()
		}else {
			tx = types.RandomTx()
		}
		c.EnQueue(tx)
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
	 hash := c.order[5]
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
	if c.cache.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	 if tx:= c.Get(tx1.GetTxHash());tx!=nil {
	 	t.Fatalf("requied nil  ;got %v",tx.String())
	 }
	 c.Remove(tx1.GetTxHash())
	 t.Log("len",c.Len())
	 c.RemoveExpired()
	t.Log("len",c.Len())
}

func TestTxCache_PopALl(t *testing.T) {
	c := newTestTxcache()
	start := time.Now()
	txs:= c.PopALl()
	t.Log("pop all used",time.Now().Sub(start).String())
	for i,tx := range txs{
		if i%1100==0 {
			t.Log(tx.String())
		}
	}
	if c.Len()!=0 {
		t.Fatalf("requied 0 , got %d",c.Len())
	}
	c = newTestTxcache()
	start  = time.Now()
	hashes := c.GetHashOrder()
	for _, hash := range hashes {
		c.remove(hash)       //10000 item 2.1s
		//c.cache.Remove(hash) // 10000 item 6ms
	}
	t.Log("remove all by hash cmp used", time.Now().Sub(start).String())

	c = newTestTxcache()
	start  = time.Now()
	for c.Len() !=0  {
		c.deQueue()     //10000 item 2.1s
		//c.cache.Remove(hash) // 10000 item 6ms
	}
	t.Log("deQueue all used", time.Now().Sub(start).String())
	c = newTestTxcache()
	start  = time.Now()
	for _, hash := range hashes {
		//c.deQueue()     //10000 item 2.1s
		c.cache.Remove(hash) // 10000 item 6ms
	}
	t.Log("directly remove  all used", time.Now().Sub(start).String())
   //result for 50000 items
   //pop all used 100ms
   //remove all by hash cmp used 28.865s
   //deQueue all used 22ms
   //directly remove  all used 22ms
}

func TestTxCache_EnQueue(t *testing.T) {
	c:= newTestTxcache()
	var wg sync.WaitGroup
	wg.Add(1)
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
	fmt.Println(c.cache.Len(),len(c.order),"used",time.Now().Sub(start))
}
