package types

import (
	"bytes"
	"encoding/binary"
)

type BinaryWriter struct {
	buf *bytes.Buffer
}

func NewBinaryWriter() *BinaryWriter {
	return &BinaryWriter{
		buf: &bytes.Buffer{},
	}
}

func (s *BinaryWriter) Write(datas ...interface{}) {
	if s.buf == nil {
		s.buf = &bytes.Buffer{}
	}
	for _, data := range datas {
		panicIfError(binary.Write(s.buf, binary.BigEndian, data))
	}
}

//get bytes
func (s *BinaryWriter) Bytes() []byte {
	return s.buf.Bytes()
}
