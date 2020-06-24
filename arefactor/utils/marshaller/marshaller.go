package marshaller

import (
	"fmt"
	"math"
)

const (
	// 0XXXXXXX
	Mfixint uint8 = 0x00

	// 111XXXXX
	Mnfixint uint8 = 0xe0

	// 1000XXXX
	Mfixmap uint8 = 0x80

	// 1001XXXX
	Mfixarray uint8 = 0x90

	// 101XXXXX
	Mfixstr uint8 = 0xa0

	Mnil      uint8 = 0xc0
	Mfalse    uint8 = 0xc2
	Mtrue     uint8 = 0xc3
	Mbin8     uint8 = 0xc4
	Mbin16    uint8 = 0xc5
	Mbin32    uint8 = 0xc6
	Mext8     uint8 = 0xc7
	Mext16    uint8 = 0xc8
	Mext32    uint8 = 0xc9
	Mfloat32  uint8 = 0xca
	Mfloat64  uint8 = 0xcb
	Muint8    uint8 = 0xcc
	Muint16   uint8 = 0xcd
	Muint32   uint8 = 0xce
	Muint64   uint8 = 0xcf
	Mint8     uint8 = 0xd0
	Mint16    uint8 = 0xd1
	Mint32    uint8 = 0xd2
	Mint64    uint8 = 0xd3
	Mfixext1  uint8 = 0xd4
	Mfixext2  uint8 = 0xd5
	Mfixext4  uint8 = 0xd6
	Mfixext8  uint8 = 0xd7
	Mfixext16 uint8 = 0xd8
	Mstr8     uint8 = 0xd9
	Mstr16    uint8 = 0xda
	Mstr32    uint8 = 0xdb
	Marray16  uint8 = 0xdc
	Marray32  uint8 = 0xdd
	Mmap16    uint8 = 0xde
	Mmap32    uint8 = 0xdf
)

var (
	HeaderSize = 5
)

// IMarshaller should meet these requirements:
// 1. MarshalMsg and UnMarshalMsg should be opposite to each other. UnMarshalMsg should
//    be able to restore the origin struct by the byte array provided by MarshalMsg.
// 2. MarshalMsg will create a byte array as a pair structure like [header - body].
//    The header part declares the size of body, the body part includes all the specific
//    info.
//    e.g. 0xcc0102
//    0xcc01 is the header, 0x02 is the body. 0xcc means it is an uint8 sized header, so
//    the 0x01 represents that the body size is 1 byte. 0x02 is the body data and its size
//    is 1.
// 3. MsgSize returns the cap size of body. It should be no less than the true body size.
type IMarshaller interface {
	MarshalMsg() ([]byte, error)
	UnMarshalMsg([]byte) ([]byte, error)
	MsgSize() int
}

func CalIMarshallerSize(imSize int) int {
	// 1 for header lead
	sz := 1

	if imSize <= math.MaxUint8 {
		sz += 1
	} else if imSize <= math.MaxUint16 {
		sz += 2
	} else if imSize <= math.MaxUint32 {
		sz += 4
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	sz += imSize
	return sz
}

func InitIMarshallerBytes(msgSize int) []byte {
	headerLen := 0
	if msgSize <= math.MaxUint8 {
		headerLen = 2
	} else if msgSize <= math.MaxUint16 {
		headerLen = 3
	} else if msgSize <= math.MaxUint32 {
		headerLen = 5
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	return make([]byte, headerLen+msgSize)
}

func EncodeHeader(b []byte, pos int, size int) ([]byte, int) {

	if size <= math.MaxUint8 {
		b[pos] = Muint8
		pos += 1
		b[pos] = uint8(size)
		pos += 1
	} else if size <= math.MaxUint16 {
		b[pos] = Muint16
		pos += 1
		SetUInt16(b, pos, uint16(size))
		pos += 2
	} else if size <= math.MaxUint32 {
		b[pos] = Muint32
		pos += 1
		SetUInt32(b, pos, uint32(size))
		pos += 4
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	return b, pos
}

func DecodeHeader(b []byte) ([]byte, int, error) {
	lead := b[0]
	sz := 0
	switch lead {
	case Muint8:
		sz = int(b[1])
		b = b[2:]
	case Muint16:
		sz = int(GetUInt16(b, 1))
		b = b[3:]
	case Muint32:
		sz = int(GetUInt32(b, 1))
		b = b[5:]
	default:
		return b, 0, fmt.Errorf("unknown lead: %x", lead)
	}

	if len(b) < sz {
		return nil, 0, fmt.Errorf("msg is incompleted, should be len: %d, get: %d", sz, len(b))
	}
	return b, sz, nil
}

// InitHeader create a header based on msg raw size
func InitHeader(msgSizeRaw int) []byte {
	b := make([]byte, 0)
	return AppendHeader(b, msgSizeRaw)
}

// AppendHeader append the header to a given byte array
func AppendHeader(b []byte, size int) []byte {
	if size <= math.MaxUint8 {
		b = append(b, Muint8)
		b = append(b, uint8(size))
	} else if size <= math.MaxUint16 {
		u16 := make([]byte, 2)
		SetUInt16(u16, 0, uint16(size))

		b = append(b, Muint16)
		b = append(b, u16...)

	} else if size <= math.MaxUint32 {
		u32 := make([]byte, 4)
		SetUInt32(u32, 0, uint32(size))

		b = append(b, Muint32)
		b = append(b, u32...)
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	return b
}

// FillHeaderData fills the header data into a byte array
func FillHeaderData(b []byte) []byte {
	header := InitHeader(len(b) - HeaderSize)
	b = b[HeaderSize-len(header):]
	copy(b[0:len(header)], header)

	return b
}

//func EncodeStr(s string) []byte {
//
//}
//
//func DecodeStr(b []byte) (string, []byte) {
//	if !isLenAndPrefixCorrect(b, mfixstr) {
//		return "", b
//	}
//
//	strLen := types.GetInt32(b, 1)
//
//	return
//}
//
//func DecodeInt32(b []byte) (int32, []byte) {
//	if !isLenAndPrefixCorrect(b, mint32) {
//		return 0, b
//	}
//	return decodeInt32WithoutPrefix(b)
//}
//
//func decodeInt32WithoutPrefix(b []byte) (int32, []byte) {
//	i := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24
//	b = b[4:]
//	return i, b
//}
//
//func isLenAndPrefixCorrect(b []byte, prefix byte) bool {
//	return len(b) != 0 && b[0] == prefix && len(b) >= mByteLen[prefix]
//}
