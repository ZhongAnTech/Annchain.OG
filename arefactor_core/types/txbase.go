package types

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"golang.org/x/crypto/sha3"
)

//go:generate msgp
var CanRecoverPubFromSig bool

type TxBaseType uint8

//add tx types here
const (
	TxBaseTypeNormal TxBaseType = iota
	TxBaseTypeSequencer
	TxBaseTypeCampaign
	TxBaseTypeTermChange
	TxBaseTypeArchive
	TxBaseAction
)

func (t TxBaseType) String() string {
	switch t {
	case TxBaseTypeNormal:
		return "TX"
	case TxBaseTypeSequencer:
		return "SQ"
	case TxBaseTypeCampaign:
		return "CP"
	case TxBaseTypeTermChange:
		return "TC"
	case TxBaseTypeArchive:
		return "AC"
	case TxBaseAction:
		return "ATX"
	default:
		return "NA"
	}

}

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
// | Seq  | Id                |                1 |                     |                 |
// | Seq  | ContractHashOrder |                1 |                     |                 |

//msgp:tuple TxBase
type TxBase struct {
	Type         TxBaseType
	Hash         common.Hash
	ParentsHash  common.Hashes
	AccountNonce uint64
	Height       uint64
	PublicKey    PublicKey //
	Signature    hexutil.Bytes
	MineNonce    uint64
	Weight       uint64
	inValid      bool
	Version      byte
	verified     verifiedType
}

//msgp:tuple TxBaseJson
type TxBaseJson struct {
	Type         TxBaseType    `json:"type"`
	Hash         common.Hash   `json:"hash"`
	ParentsHash  common.Hashes `json:"parents_hash"`
	AccountNonce uint64        `json:"account_nonce"`
	Height       uint64        `json:"height"`
	PublicKey    PublicKey     `json:"public_key"`
	Signature    hexutil.Bytes `json:"signature"`
	MineNonce    uint64        `json:"mine_nonce"`
	Weight       uint64        `json:"weight"`
	inValid      bool          `json:"in_valid"`
	Version      byte          `json:"version"`
	verified     verifiedType  `json:"-"`
}

func (t *TxBase) ToSmallCase() *TxBaseJson {
	if t == nil {
		return nil
	}
	b := TxBaseJson{
		Type:         t.Type,
		Height:       t.Height,
		Hash:         t.Hash,
		ParentsHash:  t.ParentsHash,
		AccountNonce: t.AccountNonce,
		PublicKey:    t.PublicKey,
		Signature:    t.Signature,
		MineNonce:    t.MineNonce,
		Weight:       t.Weight,
		inValid:      t.inValid,
	}
	return &b
}

func (t *TxBase) ToSmallCaseJson() ([]byte, error) {
	b := t.ToSmallCase()
	return json.Marshal(b)
}

func (t *TxBase) SetInValid(b bool) {
	t.inValid = b
}

func (t *TxBase) IsVerified() verifiedType {
	return t.verified
}

func (t *TxBase) SetVerified(v verifiedType) {
	t.verified = t.verified.merge(v)
}

func (t *TxBase) InValid() bool {
	return t.inValid
}

func (t *TxBase) GetType() TxBaseType {
	return t.Type
}

func (t *TxBase) GetHeight() uint64 {
	return t.Height
}

func (t *TxBase) GetWeight() uint64 {
	return t.Weight
}

func (t *TxBase) GetTxHash() common.Hash {
	return t.Hash
}

func (t *TxBase) GetNonce() uint64 {
	return t.AccountNonce
}

func (t *TxBase) Parents() common.Hashes {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash common.Hash) {
	t.Hash = hash
}

func (t *TxBase) String() string {
	return fmt.Sprintf("%d-[%.10s]-%dw", t.Height, t.GetTxHash().Hex(), t.Weight)
}

func (t *TxBase) CalcTxHash() (hash common.Hash) {
	w := NewBinaryWriter()

	for _, ancestor := range t.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	w.Write(t.Weight)
	w.Write(t.CalcMinedHash().Bytes)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (t *TxBase) CalcMinedHash() (hash common.Hash) {
	w := NewBinaryWriter()
	if !CanRecoverPubFromSig {
		w.Write(t.PublicKey)
	}
	w.Write(t.Signature, t.MineNonce)
	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

//CalculateWeight  a core algorithm for tx sorting,
//a tx's weight must bigger than any of it's parent's weight  and bigger than any of it's elder transaction's
func (t *TxBase) CalculateWeight(parents Txis) uint64 {
	var maxWeight uint64
	for _, p := range parents {
		if p.GetWeight() > maxWeight {
			maxWeight = p.GetWeight()
		}
	}
	return maxWeight + 1
}

func (t *TxBase) GetVersion() byte {
	return t.Version
}
