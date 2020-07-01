package og_interface

import (
	"fmt"
	"github.com/annchain/OG/arefactor/utils/marshaller"
)

const (
	FlagAddressNone byte = iota
	FlagAddress20
)

func MarshalAddress(addr Address) ([]byte, error) {
	return addr.MarshalMsg()
}

func UnmarshalAddress(b []byte) (Address, []byte, error) {
	var addr Address

	if len(b) == 0 {
		return nil, nil, marshaller.ErrorNotEnoughBytes(1, 0)
	}

	lead := b[0]
	switch lead {
	case FlagAddress20:
		addr = &Address20{}
	default:
		return nil, nil, fmt.Errorf("unknown Address lead: %x", lead)
	}

	b, err := addr.UnmarshalMsg(b)
	return addr, b, err
}
