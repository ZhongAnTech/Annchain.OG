package token

import "fmt"

type TokenID int32

const (
	KeyPrefix string = "tkID"
)

// token id
var (
	OGTokenID = int32(0)
)

func TokenTrieKey(tokenID int32) []byte {
	keyStr := KeyPrefix + fmt.Sprintf("%d", tokenID)
	return []byte(keyStr)
}

func LatestTokenIDTrieKey() []byte {
	keyStr := KeyPrefix + "latest"
	return []byte(keyStr)
}
