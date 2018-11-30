package types

import (
	"fmt"
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
		if i < 25 && !ok {
			t.Fatal("shoud be true")
		}
		fmt.Println(i, str, ok, err)
	}
	fmt.Println(m.Filter.filter)
}
