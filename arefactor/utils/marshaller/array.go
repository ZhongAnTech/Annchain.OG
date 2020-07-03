package marshaller

import (
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

func AppendIMarshallerArray(b []byte, arr []IMarshaller) ([]byte, error) {
	data := make([]byte, HeaderSize)
	for _, element := range arr {
		eleBytes, err := element.MarshalMsg()
		if err != nil {
			return nil, err
		}
		data = append(data, eleBytes...)
	}
	b = FillHeaderDataNum(b, len(arr))

	return append(b, data...), nil
}

func AppendBytesArray(b []byte, arr [][]byte) ([]byte, error) {
	data := make([]byte, HeaderSize)

	for _, bts := range arr {
		data = AppendBytes(data, bts)
	}
	data = FillHeaderDataNum(data, len(arr))

	return append(b, data...), nil
}

func ReadBytesArray(b []byte) ([][]byte, []byte, error) {

}

func MarshalStrArray() {

}

func MarshalIntArray() {

}

func MarshalIMarshallerArray(arr []IMarshaller) ([]byte, error) {
	// init size and header
	size, header := CalIMarshallerArrSizeAndHeader(arr)

	b := make([]byte, size)
	pos := 0

	// set lead and header
	b[0] = Mfixarray
	pos += 1
	copy(b[pos:len(header)+pos], header)
	pos += len(header)

	for _, element := range arr {
		eleBytes, err := element.MarshalMsg()
		if err != nil {
			return nil, err
		}
		endPos := pos + len(eleBytes)
		copy(b[pos:endPos], eleBytes)
		pos = endPos
	}

	return b[:pos], nil
}

// UnMarshalIMarshallerArrayHeader decode array header return array body bytes
// and the length of this array
func UnMarshalIMarshallerArrayHeader(b []byte) ([]byte, int, error) {
	if len(b) == 0 {
		return b, 0, fmt.Errorf("bytes is empty")
	}

	lead := b[0]
	if lead != Mfixarray {
		return b, 0, fmt.Errorf("byte lead is not mfixarray, get: %x", lead)
	}

	return DecodeHeader(b[1:])
}

func CalIMarshallerArrSizeAndHeader(arr []IMarshaller) (int, []byte) {
	// init size for array lead byte
	size := 1

	// add header len
	header := make([]byte, msgp.ArrayHeaderSize)
	header, headerLen := EncodeHeader(header, 0, len(arr))
	size += headerLen
	// add element size
	for _, ele := range arr {
		size += CalIMarshallerSize(ele.MsgSize())
	}

	return size, header[:headerLen]
}
