package marshaller_test

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"math"
	"testing"
)

type TestIMarshaller struct {
	a uint64
	b byte
	c []byte
}

func NewTestIMarshaller() *TestIMarshaller {
	return &TestIMarshaller{
		a: 0,
		b: 0x11,
		c: []byte{0x01, 0x02, 0x03},
	}
}

func (t *TestIMarshaller) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)

	b = marshaller.AppendUint64(b, t.a)
	b = append(b, t.b)
	b = marshaller.AppendBytes(b, t.c)

	b = marshaller.FillHeaderData(b)
	return b, nil
}

func (t *TestIMarshaller) UnMarshalMsg(b []byte) ([]byte, error) {
	b, size, err := marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}
	if len(b) < size {
		return nil, fmt.Errorf("msg is incompleted, should be len: %d, get: %d", size, len(b))
	}

	t.a, b, err = marshaller.ReadUint64Bytes(b)
	if err != nil {
		return nil, err
	}

	t.b = b[0]
	b = b[1:]

	t.c, b, err = marshaller.ReadBytes(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (t *TestIMarshaller) MsgSize() int {
	return marshaller.Uint64Size + 1 + marshaller.CalIMarshallerSize(len(t.c))
}

var (
	i8         = math.MaxUint8
	i16        = math.MaxUint16
	i32        = math.MaxUint32
	i64 uint64 = math.MaxUint64
)

func TestCalIMarshallerSize(t *testing.T) {
	sz8 := marshaller.CalIMarshallerSize(i8)
	if sz8 != 2+i8 {
		t.Errorf("size should be %d, get: %d", i8+2, sz8)
	}

	sz16 := marshaller.CalIMarshallerSize(i16)
	if sz16 != 3+i16 {
		t.Errorf("size should be %d, get: %d", i16+3, sz16)
	}

	sz32 := marshaller.CalIMarshallerSize(i32)
	if sz32 != 5+i32 {
		t.Errorf("size should be %d, get: %d", i32+5, sz32)
	}
}

func TestEncodeHeader(t *testing.T) {

	b := make([]byte, 10)
	b, pos := marshaller.EncodeHeader(b, 0, i8)
	if pos != 2 {
		t.Errorf("pos should be: %d, get: %d", 2, pos)
	}
	if b[0] != marshaller.Muint8 {
		t.Errorf("first byte should be muint8: %x, get: %x", marshaller.Muint8, b[0])
	}
	if hex.EncodeToString(b[1:2]) != "ff" {
		t.Errorf("header value should be: %s, get: %s", "ff", hex.EncodeToString(b[1:2]))
	}

	b = make([]byte, 10)
	b, pos = marshaller.EncodeHeader(b, 0, i16)
	if pos != 3 {
		t.Errorf("pos should be: %d, get: %d", 3, pos)
	}
	if b[0] != marshaller.Muint16 {
		t.Errorf("first byte should be muint16: %x, get: %x", marshaller.Muint16, b[0])
	}
	if hex.EncodeToString(b[1:3]) != "ffff" {
		t.Errorf("header value should be: %s, get: %s", "ffff", hex.EncodeToString(b[1:3]))
	}

	b = make([]byte, 10)
	b, pos = marshaller.EncodeHeader(b, 0, i32)
	if pos != 5 {
		t.Errorf("pos should be: %d, get: %d", 5, pos)
	}
	if b[0] != marshaller.Muint32 {
		t.Errorf("first byte should be muint32: %x, get: %x", marshaller.Muint32, b[0])
	}
	if hex.EncodeToString(b[1:5]) != "ffffffff" {
		t.Errorf("header value should be: %s, get: %s", "ffffffff", hex.EncodeToString(b[1:5]))
	}

}

func TestDecodeHeader(t *testing.T) {

	b := make([]byte, 10)
	b, pos := marshaller.EncodeHeader(b, 0, i8)
	b, sz, err := marshaller.DecodeHeader(b)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(b) != 10-pos {
		t.Errorf("return bytes is not correctly pruned. len should be: %d, get: %d", len(b), 10-pos)
	}
	if sz != i8 {
		t.Errorf("size not correct, should be: %d, get: %d", i8, sz)
	}

	b = make([]byte, 10)
	b, pos = marshaller.EncodeHeader(b, 0, i16)
	b, sz, err = marshaller.DecodeHeader(b)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(b) != 10-pos {
		t.Errorf("return bytes is not correctly pruned. len should be: %d, get: %d", len(b), 10-pos)
	}
	if sz != i16 {
		t.Errorf("size not correct, should be: %d, get: %d", i16, sz)
	}

	b = make([]byte, 10)
	b, pos = marshaller.EncodeHeader(b, 0, i32)
	b, sz, err = marshaller.DecodeHeader(b)
	if err != nil {
		t.Errorf(err.Error())
	}
	if len(b) != 10-pos {
		t.Errorf("return bytes is not correctly pruned. len should be: %d, get: %d", len(b), 10-pos)
	}
	if sz != i32 {
		t.Errorf("size not correct, should be: %d, get: %d", i32, sz)
	}

}

func TestAppendHeader(t *testing.T) {

	b := make([]byte, 0)
	b = marshaller.AppendHeader(b, i8)
	if hex.EncodeToString(b) != "ccff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "ccff", hex.EncodeToString(b))
	}

	b = make([]byte, 0)
	b = marshaller.AppendHeader(b, i16)
	if hex.EncodeToString(b) != "cdffff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "cdffff", hex.EncodeToString(b))
	}

	b = make([]byte, 0)
	b = marshaller.AppendHeader(b, i32)
	if hex.EncodeToString(b) != "ceffffffff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "ceffffffff", hex.EncodeToString(b))
	}

}

func TestFillHeaderData(t *testing.T) {
	var b []byte

	b = make([]byte, i8+marshaller.HeaderSize)
	b = marshaller.FillHeaderData(b)
	if len(b) != i8+2 {
		t.Errorf("len incorrect, should be: %d, get: %d", i8+2, len(b))
	}
	if hex.EncodeToString(b[:2]) != "ccff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "ccff", hex.EncodeToString(b[:2]))
	}

	b = make([]byte, i16+marshaller.HeaderSize)
	b = marshaller.FillHeaderData(b)
	if len(b) != i16+3 {
		t.Errorf("len incorrect, should be: %d, get: %d", i16+3, len(b))
	}
	if hex.EncodeToString(b[:3]) != "cdffff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "cdffff", hex.EncodeToString(b[:3]))
	}

	b = make([]byte, i32+marshaller.HeaderSize)
	b = marshaller.FillHeaderData(b)
	if len(b) != i32+5 {
		t.Errorf("len incorrect, should be: %d, get: %d", i32+5, len(b))
	}
	if hex.EncodeToString(b[:5]) != "ceffffffff" {
		t.Errorf("header incorrect, should be: %s, get: %s", "ceffffffff", hex.EncodeToString(b[:5]))
	}

}
