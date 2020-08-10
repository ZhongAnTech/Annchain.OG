package og_interface

import (
	"fmt"

	"github.com/annchain/OG/arefactor/utils/marshaller"
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
