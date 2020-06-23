package marshaller_test

import (
	"go_test/marshaller/marshaller"
	"testing"
)

func TestCalIMarshallerArrSizeAndHeader(t *testing.T) {
	arrLen := 100
	arr := make([]marshaller.IMarshaller, 0)
	for i := 0; i < arrLen; i++ {
		ele := NewTestIMarshaller()
		arr = append(arr, ele)
	}

	//sz, header := marshaller.CalIMarshallerArrSizeAndHeader(arr)

	// TODO

}
