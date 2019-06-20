package msg

import (
	"encoding/hex"
	"encoding/json"
)

// MarshalText implements encoding.TextMarshaler
func (b Bytes) MarshalText() ([]byte, error) {
	result := make([]byte, len(b)*2+2)
	copy(result, `0x`)
	hex.Encode(result[2:], b)
	return result, nil
}

func (b Bytes) MarshalJson() ([]byte, error) {
	s := b.String()
	return json.Marshal(&s)
}

// String returns the hex encoding of b.
func (b Bytes) String() string {
	return Encode(b)
}

// Encode encodes b as a hex string with 0x prefix.
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}
