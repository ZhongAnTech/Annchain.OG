package consensus

import (
	"crypto/sha256"
	"github.com/annchain/OG/arefactor/common/hexutil"
)

type SHA256Hasher struct {
}

func (S SHA256Hasher) Hash(s string) string {
	h := sha256.New()
	data := []byte(s)
	h.Write(data)
	sum := h.Sum(nil)
	return hexutil.ToHex(sum)
}
