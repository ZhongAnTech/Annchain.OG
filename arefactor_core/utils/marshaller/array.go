package marshaller

import (
	"github.com/annchain/OG/arefactor_core/types"
	"github.com/tinylib/msgp/msgp"
)

func MarshalStrArray() {

}

func MarshalIntArray() {

}

func MarshalIMarshallerArray(arr []IMarshaller) ([]byte, error) {
	// get total size
	var size int
	if len(arr) == 0 {
		size = msgp.ArrayHeaderSize
	} else {
		size = msgp.ArrayHeaderSize
		for _, ele := range arr {
			size += calIMarshallerSize(ele)
		}
	}

	b := make([]byte, size)

	// set 5 bytes array header
	b[0] = mfixarray
	types.SetUInt32(b, 1, uint32(size))

	pos := 5
	for _, element := range arr {
		msgSize := element.MsgSize()
		b, pos = encodeIMarshallerHeader(b, pos, msgSize)
		eleBytes, err := element.MarshalMsg()
		if err != nil {
			return nil, err
		}
		endPos := pos + msgSize
		copy(b[pos:endPos], eleBytes)
		pos = endPos
	}

	return b, nil
}

func UnMarshalIMarshallerArrayHeader(b []byte) ([]byte, int, error) {
	return decodeIMarshallerHeader(b)
}
