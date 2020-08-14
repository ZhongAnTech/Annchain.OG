package og_interface

import (
	"fmt"
	"math/big"

	"github.com/annchain/commongo/marshaller"
)

const (
	FlagHashNone byte = iota
	FlagHash32
)

func MarshalHash(hash Hash) ([]byte, error) {
	return hash.MarshalMsg()
}

func UnmarshalHash(b []byte) (Hash, []byte, error) {
	var h Hash

	// get lead
	if len(b) == 0 {
		return nil, nil, marshaller.ErrorNotEnoughBytes(1, 0)
	}
	lead := b[0]
	switch lead {
	case FlagHash32:
		h = &Hash32{}
	default:
		return nil, nil, fmt.Errorf("unknown Hash lead: %x", lead)
	}

	b, err := h.UnmarshalMsg(b)
	return h, b, err
}

func BigToHash(b *big.Int, hashFlag byte) Hash {
	switch hashFlag {
	case FlagHash32:
		return BigToHash32(b)
	default:
		return BigToHash32(b)
	}
}
