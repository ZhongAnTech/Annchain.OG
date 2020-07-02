package marshaller

import "fmt"

const (
	ByteFalse = 0x00
	ByteTrue = 0x01
)

func AppendBool(b []byte, isTrue bool) []byte {
	if isTrue {
		return append(b, ByteTrue)
	} else {
		return append(b, ByteFalse)
	}
}

func ReadBool(b []byte) (bool, []byte, error) {
	if len(b) == 0 {
		return false, b, ErrorNotEnoughBytes(1, 0)
	}

	bt := b[0]
	if bt == ByteTrue {
		return true, b[1:], nil
	} else if bt == ByteFalse {
		return false, b[1:], nil
	} else {
		return false, b, fmt.Errorf("unknown bool byte, should be 0x00 or 0x01")
	}
}
