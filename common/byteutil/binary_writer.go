package byteutil

import (
	"bytes"
	"encoding/binary"
	"github.com/annchain/OG/common/utilfuncs"
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
		utilfuncs.PanicIfError(binary.Write(s.buf, binary.BigEndian, data), "binary writer")
	}
}

//get bytes
func (s *BinaryWriter) Bytes() []byte {
	return s.buf.Bytes()
}
