package types

import (
	"crypto/sha256"
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"math/rand"
	"sort"
	"testing"
	"time"
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
		if i < 25 && !ok {
			t.Fatal("should be true")
		}
		fmt.Println(i, str, ok, err)
	}
	fmt.Println(m.Filter.filter)
}

func TestRandomTx(t *testing.T) {
	var txis Txis
	for i := 0; i < 50; i++ {
		if i%10 == 0 {
			tx := RandomSequencer()
			tx.Height = uint64(rand.Intn(4))
			tx.Weight = uint64(rand.Intn(10))
			txis = append(txis, Txi(tx))
		} else {
			tx := RandomTx()
			tx.Height = uint64(rand.Intn(4))
			tx.Weight = uint64(rand.Intn(10))
			txis = append(txis, Txi(tx))
		}
	}
	fmt.Println(len(txis), txis)
	sort.Sort(txis)
	fmt.Println(len(txis), txis)
}

func TestHash_Cmp(t *testing.T) {
	m := MessageTxsResponse{}
	for i := 0; i < 10000; i++ {
		tx := RandomTx()
		m.RawTxs = append(m.RawTxs, tx.RawTx())
	}
	data1, _ := m.MarshalMsg(nil)
	h := sha256.New()
	start := time.Now()
	h.Write(data1)
	s1 := h.Sum(nil)
	fmt.Println(time.Now().Sub(start), "used for txs ", "len", len(data1), hexutil.Encode(s1))
	m2 := MessageNewTx{
		RawTx: RandomTx().RawTx(),
	}
	data2, _ := m2.MarshalMsg(nil)
	h = sha256.New()
	start = time.Now()
	h.Write(data2)
	s2 := h.Sum(nil)
	fmt.Println(time.Now().Sub(start), "used for tx", "len", len(data2), hexutil.Encode(s2))
}

func TestRawTxs_String(t *testing.T) {
	var r RawSequencers
	fmt.Println(r)
	var s MessageSyncResponse
	fmt.Println(s)
}
