package edwards25519

import (
	"go.dedis.ch/kyber/v3"
	"io"
)

func UnmarshalBinaryScalar(buf []byte) (kyber.Scalar, error) {
	var s scalar
	err := s.UnmarshalBinary(buf)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func UnmarshalFromScalar(r io.Reader) (kyber.Scalar, int, error) {
	var s scalar
	n, err := s.UnmarshalFrom(r)
	if err != nil {
		return nil, n, err
	}
	return &s, n, nil
}

func UnmarshalBinaryPoint(b []byte) (kyber.Point, error) {
	var P point
	if err := P.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return &P, nil
}

func UnmarshalFromPoint(r io.Reader) (kyber.Point, int, error) {
	var P point
	n, err := P.UnmarshalFrom(r)
	if err != nil {
		return nil, n, err
	}
	return &P, n, nil
}