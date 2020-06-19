package marshaller

import (
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

func MarshalStrArray() {

}

func MarshalIntArray() {

}

func MarshalIMarshallerArray(arr []IMarshaller) ([]byte, error) {
	// get total size

	// init size for array lead byte
	header := make([]byte, msgp.ArrayHeaderSize)
	header, pos := EncodeIMarshallerHeader(header, 1, len(arr))
	// add header len
	size := pos
	// add element size
	for _, ele := range arr {
		size += calIMarshallerSize(ele)
	}

	b := make([]byte, size)

	// set lead and header
	b[0] = mfixarray
	copy(b[1:pos], header)

	for _, element := range arr {
		eleBytes, err := element.MarshalMsg()
		if err != nil {
			return nil, err
		}
		endPos := pos + len(eleBytes)
		copy(b[pos:endPos], eleBytes)
		pos = endPos
	}

	return b, nil
}

// UnMarshalIMarshallerArrayHeader decode array header return array body bytes
// and the length of this array
func UnMarshalIMarshallerArrayHeader(b []byte) ([]byte, int, error) {
	if len(b) == 0 {
		return b, 0, fmt.Errorf("bytes is empty")
	}

	lead := b[0]
	if lead != mfixarray {
		return b, 0, fmt.Errorf("byte lead is not mfixarray, get: %x", lead)
	}

	return DecodeIMarshallerHeader(b[1:])
}
