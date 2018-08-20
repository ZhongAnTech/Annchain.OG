package types

import (
	"fmt"
	"golang.org/x/crypto/sha3"
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

// RUN "go generate" to generate all help codes

//go:generate msgp
//msgp:tuple Foo

type Hash []byte

type Foo struct {
	Bar     string             `msg:"bar"`
	Baz     float64            `msg:"baz"`
	Address Hash               `msg:"address"`
	Parents []Hash             `msg:"parents"`
	KV      map[string]float64 `msg:"kv"`
	//BIG *big.Int `msg:"big"`
}

func (f *Foo) CalcHash() {
	var buf bytes.Buffer

	//panicIfError(binary.Write(&buf, binary.LittleEndian, f.Bar))
	panicIfError(binary.Write(&buf, binary.LittleEndian, f.Baz))
	panicIfError(binary.Write(&buf, binary.LittleEndian, f.Address))
	//panicIfError(binary.Write(&buf, binary.LittleEndian, f.Parents))
	//panicIfError(binary.Write(&buf, binary.LittleEndian, f.KV))

	hasher := sha3.New512()
	v := hasher.Sum(buf.Bytes())
	fmt.Println(hex.Dump(v))

}
func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}
