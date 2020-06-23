package marshaller_test

import (
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"testing"
)

func TestAppendBytes(t *testing.T) {
	var b, bts []byte
	var err error
	var btsLen int

	b = make([]byte, 0)
	btsLen = i8
	bts = make([]byte, btsLen)
	b = marshaller.AppendBytes(b, bts)
	bts, b, err = marshaller.ReadBytes(b)
	if err != nil {
		t.Errorf("read bytes error: %v", err)
	}
	if len(bts) != btsLen {
		t.Errorf("bytes len incorrect, should be: %d, get: %d", btsLen, len(bts))
	}

	b = make([]byte, 0)
	btsLen = i16
	bts = make([]byte, btsLen)
	b = marshaller.AppendBytes(b, bts)
	bts, b, err = marshaller.ReadBytes(b)
	if err != nil {
		t.Errorf("read bytes error: %v", err)
	}
	if len(bts) != btsLen {
		t.Errorf("bytes len incorrect, should be: %d, get: %d", btsLen, len(bts))
	}

	b = make([]byte, 0)
	btsLen = i32
	bts = make([]byte, btsLen)
	b = marshaller.AppendBytes(b, bts)
	bts, b, err = marshaller.ReadBytes(b)
	if err != nil {
		t.Errorf("read bytes error: %v", err)
	}
	if len(bts) != btsLen {
		t.Errorf("bytes len incorrect, should be: %d, get: %d", btsLen, len(bts))
	}

}
