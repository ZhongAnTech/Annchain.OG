package marshaller

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common"
	"github.com/annchain/OG/arefactor/common/math"
)

const (
	// 0XXXXXXX
	mfixint uint8 = 0x00

	// 111XXXXX
	mnfixint uint8 = 0xe0

	// 1000XXXX
	mfixmap uint8 = 0x80

	// 1001XXXX
	mfixarray uint8 = 0x90

	// 101XXXXX
	mfixstr uint8 = 0xa0

	mnil      uint8 = 0xc0
	mfalse    uint8 = 0xc2
	mtrue     uint8 = 0xc3
	mbin8     uint8 = 0xc4
	mbin16    uint8 = 0xc5
	mbin32    uint8 = 0xc6
	mext8     uint8 = 0xc7
	mext16    uint8 = 0xc8
	mext32    uint8 = 0xc9
	mfloat32  uint8 = 0xca
	mfloat64  uint8 = 0xcb
	muint8    uint8 = 0xcc
	muint16   uint8 = 0xcd
	muint32   uint8 = 0xce
	muint64   uint8 = 0xcf
	mint8     uint8 = 0xd0
	mint16    uint8 = 0xd1
	mint32    uint8 = 0xd2
	mint64    uint8 = 0xd3
	mfixext1  uint8 = 0xd4
	mfixext2  uint8 = 0xd5
	mfixext4  uint8 = 0xd6
	mfixext8  uint8 = 0xd7
	mfixext16 uint8 = 0xd8
	mstr8     uint8 = 0xd9
	mstr16    uint8 = 0xda
	mstr32    uint8 = 0xdb
	marray16  uint8 = 0xdc
	marray32  uint8 = 0xdd
	mmap16    uint8 = 0xde
	mmap32    uint8 = 0xdf
)

var (
	HeaderSize = 5
)

type IMarshaller interface {
	MarshalMsg() ([]byte, error)
	UnMarshalMsg([]byte) ([]byte, error)
	MsgSize() int
}

func CalIMarshallerSize(im IMarshaller) int {
	// 1 for header lead
	sz := 1

	msgSize := im.MsgSize()
	if msgSize <= math.MaxUint8 {
		sz += 1
	} else if msgSize <= math.MaxUint16 {
		sz += 2
	} else if msgSize <= math.MaxUint32 {
		sz += 4
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	sz += msgSize
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
		b[pos] = muint8
		pos += 1
		b[pos] = uint8(size)
		pos += 1
	} else if size <= math.MaxUint16 {
		b[pos] = muint16
		pos += 1
		common.SetUInt16(b, pos, uint16(size))
		pos += 2
	} else if size <= math.MaxUint32 {
		b[pos] = muint32
		pos += 1
		common.SetUInt32(b, pos, uint32(size))
		pos += 4
	} else {
		// size should not be larger than 2^32-1
		panic("size should less than 2^32-1")
	}

	return b, pos
}

func DecodeHeader(b []byte) ([]byte, int, error) {
	lead := b[0]
	switch lead {
	case muint8:
		return b[2:], int(b[1]), nil
	case muint16:
		sz := common.GetUInt16(b, 1)
		return b[3:], int(sz), nil
	case muint32:
		sz := common.GetUInt32(b, 1)
		return b[5:], int(sz), nil
	default:
		return b, 0, fmt.Errorf("unknown lead: %x", lead)
	}

}

// InitHeader create a header based on msg raw size
func InitHeader(msgSizeRaw int) []byte {
	b := make([]byte, 0)
	return AppendHeader(b, msgSizeRaw)
}

// AppendHeader append the header to a given byte array
func AppendHeader(b []byte, size int) []byte {
	if size <= math.MaxUint8 {
		b = append(b, muint8)
		b = append(b, uint8(size))
	} else if size <= math.MaxUint16 {
		u16 := make([]byte, 2)
		common.SetUInt16(u16, 0, uint16(size))

		b = append(b, muint16)
		b = append(b, u16...)

	} else if size <= math.MaxUint32 {
		u32 := make([]byte, 4)
		common.SetUInt32(u32, 0, uint32(size))

		b = append(b, muint32)
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
