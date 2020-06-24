package hexutil

import "encoding/hex"

// FromHex returns the bytes represented by the hexadecimal string s.
func FromHex(s string) ([]byte, error) {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return hex.DecodeString(s)
}

func ToHex(b []byte) string {
	return ToHexWithPad(b, 0)
}

func ToHexWithPad(b []byte, padLength int) string {
	hexstr := hex.EncodeToString(b)
	if len(hexstr) == 0 {
		hexstr = "0"
	}
	for len(hexstr) < padLength {
		hexstr = "0" + hexstr
	}
	return hexstr
}

func ToFormalHex(b []byte) string {
	return "0x" + ToHex(b)
}

func ToFormalHexWithPad(b []byte, padLength int) string {
	return "0x" + ToHexWithPad(b, padLength)
}

func ToBriefHex(bytes []byte, maxLen int) string {
	if maxLen >= len(bytes) {
		return hex.EncodeToString(bytes)
	}
	return hex.EncodeToString(bytes[0:maxLen/2]) + "..." + hex.EncodeToString(bytes[maxLen/2:len(bytes)])
}
