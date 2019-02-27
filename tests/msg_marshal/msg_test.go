package msg_marshal

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestPerson_MarshalMsg(t *testing.T) {
	p := Person{
		Name: "alice",
		Age:  10,
		Type: 1,
	}
	p2 := Person{
		Name: "bob",
		Age:  1522,
		Type: 2,
	}
	s1 := Student{p, 15}
	t1 := Teacher{p2, true}
	var fooi FooI
	fooi = &s1
	data, _ := MashalFoo(fooi, nil)
	fmt.Println(hex.EncodeToString(data))
	_, s2, err := UnmarShalFoo(data)
	fmt.Println(s2, err)
	if s1 != *s2.(*Student) {
		t.Fail()
	}
	_ = t1
}
