package math

import (
	"testing"
	"fmt"
)

func TestMoney(t *testing.T) {
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	a, success := NewBigIntFromString(s, 10)
	if !success {
		panic(s)
	}

	a.SetString(s, 10)
	a2 := NewBigInt(2)
	a3 := NewBigIntFromBigInt(a.bigint.Add(a.bigint, a2.bigint))

	fmt.Println(a)
	fmt.Println(a2)
	fmt.Println(a3)

	a3 = NewBigIntFromBigInt(a.bigint.Add(a.bigint, a2.bigint))

	fmt.Println(a3)
}
