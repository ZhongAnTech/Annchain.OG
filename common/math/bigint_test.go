package math

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
	"encoding/hex"
)

func TestMoney(t *testing.T) {
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	a, success := NewBigIntFromString(s, 10)
	if !success {
		panic(s)
	}

	a.SetString(s, 10)
	a2 := NewBigInt(2)
	a3 := NewBigIntFromBigInt(a.Value.Add(a.Value, a2.Value))

	fmt.Println(a)
	fmt.Println(a2)
	fmt.Println(a3)

	a3 = NewBigIntFromBigInt(a.Value.Add(a.Value, a2.Value))

	fmt.Println(a3)
}

func TestSerialization(t *testing.T) {
	s := "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	a, success := NewBigIntFromString(s, 10)
	if !success {
		panic(s)
	}
	bytes, err := a.MarshalMsg(nil)
	assert.NoError(t,err)
	hex.Dump(bytes)
	b := &BigInt{}
	bytes, err = b.UnmarshalMsg(bytes)
	assert.NoError(t,err)
	fmt.Println(b.String())
	assert.Equal(t, s, b.String())
}