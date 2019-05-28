package bn256

import (
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
	"io"
)

func UnmarshalBinaryPointG1(buf []byte) (q kyber.Point, err error) {
	var p pointG1
	err = p.UnmarshalBinary(buf)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UnmarshalBinaryPointG2(buf []byte) (q kyber.Point, err error) {
	var p pointG2
	err = p.UnmarshalBinary(buf)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UnmarshalFromPointG2(r io.Reader) (kyber.Point, int, error) {
	var p pointG2
	n, err := p.UnmarshalFrom(r)
	if err != nil {
		return nil, n, err
	}
	return &p, n, nil
}

func UnmarshalFromPointG1(r io.Reader) (kyber.Point, int, error) {
	var p pointG1
	n, err := p.UnmarshalFrom(r)
	if err != nil {
		return nil, n, err
	}
	return &p, n, nil
}
