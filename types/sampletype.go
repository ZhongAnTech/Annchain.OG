package types

import (
	"golang.org/x/crypto/sha3"
	"bytes"
	"encoding/binary"
	"fmt"
)

// Define your own structure and then use messagepack to generate codes
// You can add functions for your struct here.
// RUN "go generate" to generate all helper codes

// DO NOT DELETE THIS TWO COMMENTS. THEY ARE FUNCTIONAL.

//go:generate msgp
//msgp:tuple Foo

type Foo struct {
	Bar     string             `msg:"bar"`
	Baz     float64            `msg:"baz"`
	Address Address            `msg:"address"`
	Parents []Hash             `msg:"parents"`
	KV      map[string]float64 `msg:"kv"`
	Seq     Sequencer          `msg:"seq"`
	TxInner Tx                 `msg:"tx"`
	//BIG *big.Int `msg:"big"`
}

func (f *Foo) CalcHash() (hash Hash, err error) {
	var buf bytes.Buffer

	if err := binary.Write(&buf, binary.LittleEndian, f.Bar); err != nil {
		return Hash{}, err
	}

	hasher := sha3.New512()
	v := hasher.Sum(buf.Bytes())
	hash.MustSetBytes(v)
	fmt.Println(hash.Hex())
	return
}
