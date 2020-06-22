package marshaller

import "github.com/tinylib/msgp/msgp"

func AppendUint64(b []byte, u uint64) []byte {
	return msgp.AppendUint64(b, u)
}

func ReadUint64Bytes(b []byte) (u uint64, o []byte, err error) {
	return msgp.ReadUint64Bytes(b)
}
