package mod

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
)

func TestInt_MarshalID(t *testing.T) {
	i := new(big.Int).Lsh(big.NewInt(1), 128)
	m := NewInt64(17,i)
	buf := bytes.NewBuffer(nil)
	m.MarshalTo(buf)
	fmt.Println(buf.Bytes())
}
