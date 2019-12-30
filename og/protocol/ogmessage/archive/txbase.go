package archive

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
)

//go:generate msgp
var CanRecoverPubFromSig bool

type TxBaseType uint16

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
	verified     VerifiedType  `json:"-"`
}

//
//func (t *TxBase) ToSmallCase() *TxBaseJson {
//	if t == nil {
//		return nil
//	}
//	b := TxBaseJson{
//		Type:         t.Type,
//		Height:       t.Height,
//		Hash:         t.Hash,
//		ParentsHash:  t.ParentsHash,
//		AccountNonce: t.AccountNonce,
//		PublicKey:    t.PublicKey,
//		Signature:    t.Signature,
//		MineNonce:    t.MineNonce,
//		Weight:       t.Weight,
//		inValid:      t.inValid,
//	}
//	return &b
//}
//
//func (t *TxBase) ToSmallCaseJson() ([]byte, error) {
//	b := t.ToSmallCase()
//	return json.Marshal(b)
//}
//
//func (t *TxBase) SetValid(b bool) {
//	t.inValid = b
//}
//
//func (t *TxBase) IsVerified() VerifiedType {
//	return t.verified
//}
//
//func (t *TxBase) SetVerified(v VerifiedType) {
//	t.verified = t.verified.Merge(v)
//}
//
//func (t *TxBase) Valid() bool {
//	return t.inValid
//}
//
//func (t *TxBase) GetType() TxBaseType {
//	return t.Type
//}
//
//func (t *TxBase) GetHeight() uint64 {
//	return t.Height
//}
//
//func (t *TxBase) GetWeight() uint64 {
//	return t.Weight
//}
//
//func (t *TxBase) GetTxHash() common.Hash {
//	return t.Hash
//}
//
//func (t *TxBase) GetNonce() uint64 {
//	return t.AccountNonce
//}
//
//func (t *TxBase) GetParents() common.Hashes {
//	return t.ParentsHash
//}
//
//func (t *TxBase) SetHash(hash common.Hash) {
//	t.Hash = hash
//}
//
//func (t *TxBase) GetVersion() byte {
//	return t.Version
//}
