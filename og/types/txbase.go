package types

import (
	"github.com/annchain/OG/common"
	"strings"
)

// Here indicates what fields should be concerned during hash calculation and signature generation
// |      |                   | Signature target |     NonceHash(Slow) |    TxHash(Fast) |
// |------|-------------------|------------------|---------------------|-----------------|
// | Base | ParentsHash       |                  |                     |               1 |
// | Base | Height            |                  |                     |                 |
// | Base | PublicKey         |                  |                   1 | 1 (nonce hash)  |
// | Base | Signature         |                  |                   1 | 1 (nonce hash)  |
// | Base | MinedNonce        |                  |                   1 | 1 (nonce hash)  |
// | Base | AccountNonce      |                1 |                     |                 |
// | Tx   | From              |                1 |                     |                 |
// | Tx   | To                |                1 |                     |                 |
// | Tx   | Value             |                1 |                     |                 |
// | Tx   | Data              |                1 |                     |                 |
// | Seq  | MyIndex           |                1 |                     |                 |
// | Seq  | ContractHashOrder |                1 |                     |                 |

type TxBase struct {
	//	Type         TxBaseType
	Hash        common.Hash
	ParentsHash common.Hashes
	//	AccountNonce uint64
	Height uint64
	//	PublicKey    crypto.PublicKey //
	//	Signature    hexutil.KeyBytes
	//	MineNonce    uint64
	//	Weight       uint64
	//	inValid      bool
	//	Version      byte
	//	//verified     VerifiedType
}

type Txis []Txi

func (t Txis) String() string {
	var strs []string
	for _, v := range t {
		strs = append(strs, v.String())
	}
	return strings.Join(strs, ", ")
}

func (t Txis) Len() int {
	return len(t)
}

func (t Txis) Less(i, j int) bool {
	if t[i].GetWeight() < t[j].GetWeight() {
		return true
	}
	if t[i].GetWeight() > t[j].GetWeight() {
		return false
	}
	//if t[i].GetNonce() < t[j].GetNonce() {
	//	return true
	//}
	//if t[i].GetNonce() > t[j].GetNonce() {
	//	return false
	//}
	if t[i].GetTxHash().Cmp(t[j].GetTxHash()) < 0 {
		return true
	}
	return false
}

func (t Txis) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
