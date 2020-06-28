package marshaller

import (
	"github.com/tinylib/msgp/msgp"
	"math/big"
)

func AppendInt32(b []byte, i int32) []byte {
	return msgp.AppendInt32(b, i)
}

func ReadInt32Bytes(b []byte) (i int32, o []byte, err error) {
	return msgp.ReadInt32Bytes(b)
}

func AppendUint64(b []byte, u uint64) []byte {
	return msgp.AppendUint64(b, u)
}

func ReadUint64Bytes(b []byte) (u uint64, o []byte, err error) {
	return msgp.ReadUint64Bytes(b)
}

/**
BigInteger
 */

func AppendBigInt(b []byte, bi *big.Int) []byte {
	return AppendBytes(b, bi.Bytes())
}

func ReadBigInt(b []byte) (bi *big.Int, o []byte, err error) {
	bts, b, err := ReadBytes(b)
	if err != nil {
		return nil, b, err
	}
	return big.NewInt(0).SetBytes(bts), b, nil
}

func CalBigIntSize(bi *big.Int) int {
	return CalIMarshallerSize(len(bi.Bytes()))
}

/**
Getters and Readers
 */

func GetUInt16(b []byte, pos int) uint16 {
	return uint16(b[pos]) | uint16(b[pos+1])<<8
}

func SetUInt16(b []byte, pos int, i uint16) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
}

func GetInt16(b []byte, pos int) int16 {
	return int16(b[pos]) | int16(b[pos+1])<<8
}

func SetInt16(b []byte, pos int, i int16) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
}

// GetUInt32 get an uint32 from byte array with a start position.
// This is for those little-endian bytes.
func GetUInt32(b []byte, pos int) uint32 {
	return uint32(b[pos]) | uint32(b[pos+1])<<8 | uint32(b[pos+2])<<16 | uint32(b[pos+3])<<24
}

// SetUInt32 set an uint32 into byte array at a position.
func SetUInt32(b []byte, pos int, i uint32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

// GetInt32 get an int32 from byte array with a start position.
func GetInt32(b []byte, pos int) int32 {
	return int32(b[pos]) | int32(b[pos+1])<<8 | int32(b[pos+2])<<16 | int32(b[pos+3])<<24
}

// SetInt32 set an int32 into byte array at a position.
func SetInt32(b []byte, pos int, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

// GetInt64 get an int64 from byte array with a start position.
func GetInt64(b []byte, pos int) int64 {
	return int64(b[pos]) | int64(b[pos+1])<<8 | int64(b[pos+2])<<16 | int64(b[pos+3])<<24 |
		int64(b[pos+4])<<32 | int64(b[pos+5])<<40 | int64(b[pos+6])<<48 | int64(b[pos+7])<<56
}

func GetUint64(b []byte, pos int) uint64 {
	return uint64(b[pos]) | uint64(b[pos+1])<<8 | uint64(b[pos+2])<<16 | uint64(b[pos+3])<<24 |
		uint64(b[pos+4])<<32 | uint64(b[pos+5])<<40 | uint64(b[pos+6])<<48 | uint64(b[pos+7])<<56
}

func SetUint64(b []byte, pos int, i uint64) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
	b[pos+4] = byte(i >> 32)
	b[pos+5] = byte(i >> 40)
	b[pos+6] = byte(i >> 48)
	b[pos+7] = byte(i >> 56)
}

func Int32ToBytes(i int32) []byte {
	b := make([]byte, 4)

	b[0] = byte(i)
	b[1] = byte(i >> 8)
	b[2] = byte(i >> 16)
	b[3] = byte(i >> 24)

	return b
}
