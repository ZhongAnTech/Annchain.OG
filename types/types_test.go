package types

import (
	"fmt"
	"testing"
)

func TestCuckooFilter_EncodeMsg(t *testing.T) {
	m := &MessageNewTx{}
	for i := 0; i < 25; i++ {
		str := "abcdef" + fmt.Sprintf("%d%d%d", i, i+2, i) + "ef"
		err := m.AddItem([]byte(str))
		date, err := m.BloomFilter.filter.GobEncode()
		fmt.Println("len ", len(date), err)
		out, _ := m.MarshalMsg(nil)
		fmt.Println("i", i, "size ", m.Msgsize(), "len", len(out), err, len(m.BloomFilter.Data))
	}
	for i := 0; i < 37; i++ {
		str := "abcdef" + fmt.Sprintf("%d%d%d", i, i+2, i) + "ef"
		ok, err := m.LookUpItem([]byte(str))
		fmt.Println(i, str, ok, err)
	}
	fmt.Println(m.BloomFilter.filter)

}
