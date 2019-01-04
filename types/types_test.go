package types

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestCuckooFilter_EncodeMsg(t *testing.T) {
	m := &MessageSyncRequest{}
	m.Filter = NewDefaultBloomFilter()
	for i := 0; i < 25; i++ {
		str := "abcdef" + fmt.Sprintf("%d%d%d", i, i+2, i) + "ef"
		m.Filter.AddItem([]byte(str))
		date, err := m.Filter.filter.Encode()
		fmt.Println("len ", len(date), err)
		out, _ := m.MarshalMsg(nil)
		fmt.Println("i", i, "size ", m.Msgsize(), "len", len(out), err, len(m.Filter.Data))
	}
	for i := 0; i < 37; i++ {
		str := "abcdef" + fmt.Sprintf("%d%d%d", i, i+2, i) + "ef"
		ok, err := m.Filter.LookUpItem([]byte(str))
		if i< 25 && !ok {
			t.Fatal("shoud be true")
		}
		fmt.Println(i, str, ok, err)
	}
	fmt.Println(m.Filter.filter)
}

func TestRandomTx(t *testing.T) {
	var txis  Txis
	for i:= 0;i<50;i++ {
		if i%10==0 {
			tx:= RandomSequencer()
			tx.Height = uint64(rand.Intn(4))
			tx.Order = uint32(rand.Intn(10))
			txis = append(txis,Txi(tx))
		}else {
			tx:= RandomTx()
			tx.Height =uint64(rand.Intn(4))
			tx.Order = uint32(rand.Intn(10))
			txis = append(txis,Txi(tx))
		}
	}
	fmt.Println(len(txis),txis)
	sort.Sort(txis)
	fmt.Println(len(txis),txis)
}
