package og_interface

import (
	"fmt"

	"github.com/annchain/OG/arefactor/utils/marshaller"
	"github.com/annchain/commongo/hexutil"
)

const (
	FlagAddressNone byte = iota
	FlagAddress20
)

func AddressFromHex(s string) (Address, error) {
	bts, err := hexutil.FromHex(s)
	if err != nil {
		return nil, err
	}

	switch len(bts) {
	case Address20Length:
		var addr Address20
		addr.FromBytes(bts)

		return &addr, nil
	default:
		return nil, fmt.Errorf("unknown address hex length: %d", len(bts))
	}
}

func AddressFromAddressKey(key AddressKey) (Address, error) {
	return AddressFromHex(string(key))
}

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
