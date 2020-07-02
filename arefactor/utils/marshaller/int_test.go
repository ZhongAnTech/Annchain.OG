package marshaller_test

import (
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"testing"
)

func TestAppendUint64(t *testing.T) {
	var b []byte
	var u64, target uint64
	var err error

	target = uint64(i8)
	b = make([]byte, 0)
	b = marshaller.AppendUint64(b, target)
	u64, b, err = marshaller.ReadUint64(b)
	if err != nil {
		t.Errorf("Read uint64 error: %v", err)
	}
	if u64 != target {
		t.Errorf("uint64 incorrect, should be: %d, get: %d", u64, target)
	}

	target = uint64(i16)
	b = make([]byte, 0)
	b = marshaller.AppendUint64(b, target)
	u64, b, err = marshaller.ReadUint64(b)
	if err != nil {
		t.Errorf("Read uint64 error: %v", err)
	}
	if u64 != target {
		t.Errorf("uint64 incorrect, should be: %d, get: %d", u64, target)
	}

	target = uint64(i32)
	b = make([]byte, 0)
	b = marshaller.AppendUint64(b, target)
	u64, b, err = marshaller.ReadUint64(b)
	if err != nil {
		t.Errorf("Read uint64 error: %v", err)
	}
	if u64 != target {
		t.Errorf("uint64 incorrect, should be: %d, get: %d", u64, target)
	}

	target = uint64(i64)
	b = make([]byte, 0)
	b = marshaller.AppendUint64(b, target)
	u64, b, err = marshaller.ReadUint64(b)
	if err != nil {
		t.Errorf("Read uint64 error: %v", err)
	}
	if u64 != target {
		t.Errorf("uint64 incorrect, should be: %d, get: %d", u64, target)
	}

}
