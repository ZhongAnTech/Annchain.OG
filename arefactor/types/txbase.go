package types

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"github.com/annchain/OG/common/hexutil"
	"golang.org/x/crypto/sha3"

	og_types "github.com/annchain/OG/arefactor/og_interface"
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
	Hash         og_types.Hash
	ParentsHash  []og_types.Hash
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
	Type         TxBaseType      `json:"type"`
	Hash         og_types.Hash   `json:"hash"`
	ParentsHash  []og_types.Hash `json:"parents_hash"`
	AccountNonce uint64          `json:"account_nonce"`
	Height       uint64          `json:"height"`
	PublicKey    PublicKey       `json:"public_key"`
	Signature    hexutil.Bytes   `json:"signature"`
	MineNonce    uint64          `json:"mine_nonce"`
	Weight       uint64          `json:"weight"`
	inValid      bool            `json:"in_valid"`
	Version      byte            `json:"version"`
	verified     verifiedType    `json:"-"`
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

func (t *TxBase) GetTxHash() og_types.Hash {
	return t.Hash
}

func (t *TxBase) GetNonce() uint64 {
	return t.AccountNonce
}

func (t *TxBase) Parents() []og_types.Hash {
	return t.ParentsHash
}

func (t *TxBase) SetHash(hash og_types.Hash) {
	t.Hash = hash
}

func (t *TxBase) String() string {
	return fmt.Sprintf("%d-[%.10s]-%dw", t.Height, t.GetTxHash().Hex(), t.Weight)
}

func (t *TxBase) CalcTxHash() (hash og_types.Hash) {
	w := NewBinaryWriter()

	for _, ancestor := range t.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	w.Write(t.Weight)
	w.Write(t.CalcMinedHash().Bytes)

	result := sha3.Sum256(w.Bytes())
	hash.FromBytes(result[0:])
	return
}

func (t *TxBase) CalcMinedHash() (hash og_types.Hash) {
	w := NewBinaryWriter()
	if !CanRecoverPubFromSig {
		w.Write(t.PublicKey)
	}
	w.Write(t.Signature, t.MineNonce)
	result := sha3.Sum256(w.Bytes())
	hash.FromBytes(result[0:])
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

func (t *TxBase) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)
	//b, pos := marshaller.EncodeHeader(b, 0, size)

	// (byte) Type + Version
	b = append(b, byte(t.Type))
	b = append(b, t.Version)

	// (uint64) AccountNonce, Height, MineNonce, Weight
	b = marshaller.AppendUint64(b, t.AccountNonce)
	b = marshaller.AppendUint64(b, t.Height)
	b = marshaller.AppendUint64(b, t.MineNonce)
	b = marshaller.AppendUint64(b, t.Weight)

	// ([]byte) PublicKey, Signature
	b = marshaller.AppendBytes(b, t.PublicKey)
	b = marshaller.AppendBytes(b, t.Signature)

	// (Hash) Hash
	hashB, err := t.Hash.MarshalMsg()
	if err != nil {
		return nil, err
	}
	b = append(b, hashB...)

	// ([]Hash) ParentsHash
	hashes := make([]marshaller.IMarshaller, 0)
	for _, hash := range t.ParentsHash {
		hashes = append(hashes, hash)
	}
	hashesB, err := marshaller.MarshalIMarshallerArray(hashes)
	if err != nil {
		return nil, err
	}
	b = append(b, hashesB...)

	// fill in header bytes
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (t *TxBase) UnMarshalMsg(b []byte) ([]byte, error) {
	b, size, err := marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}
	if len(b) < size {
		return nil, fmt.Errorf("msg is incompleted, should be len: %d, get: %d", size, len(b))
	}

	// Type, Version
	t.Type = TxBaseType(b[0])
	t.Version = b[1]
	b = b[2:]

	// (uint64) AccountNonce, Height, MineNonce, Weight
	t.AccountNonce, b, err = marshaller.ReadUint64Bytes(b)
	if err != nil {
		return nil, err
	}
	t.Height, b, err = marshaller.ReadUint64Bytes(b)
	if err != nil {
		return nil, err
	}
	t.MineNonce, b, err = marshaller.ReadUint64Bytes(b)
	if err != nil {
		return nil, err
	}
	t.Weight, b, err = marshaller.ReadUint64Bytes(b)
	if err != nil {
		return nil, err
	}

	// ([]byte) PublicKey, Signature
	t.PublicKey, b, err = marshaller.ReadBytes(b)
	if err != nil {
		return nil, err
	}
	t.Signature, b, err = marshaller.ReadBytes(b)
	if err != nil {
		return nil, err
	}

	// Hash
	t.Hash, b, err = og_types.UnmarshalHash(b)
	if err != nil {
		return nil, err
	}

	// ParentsHash
	b, arrLen, err := marshaller.UnMarshalIMarshallerArrayHeader(b)
	if err != nil {
		return nil, fmt.Errorf("unmarshal tx hashes arr header error: %v", err)
	}
	t.ParentsHash = make([]og_types.Hash, 0)
	for i := 0; i < arrLen; i++ {
		var hash og_types.Hash
		hash, b, err = og_types.UnmarshalHash(b)
		if err != nil {
			return nil, err
		}
		t.ParentsHash = append(t.ParentsHash, hash)
	}

	return b, nil
}

func (t *TxBase) MsgSize() int {
	size := 0
	// t.Type + t.Hash
	size += 1 + marshaller.CalIMarshallerSize(t.Hash)
	// t.ParentsHash
	hashesI := make([]marshaller.IMarshaller, 0)
	for _, h := range t.ParentsHash {
		hashesI = append(hashesI, h)
	}
	hashesSize, _ := marshaller.CalIMarshallerArrSizeAndHeader(hashesI)
	size += hashesSize
	// t.AccountNonce + t.Height + t.MineNonce + t.Weight + PublicKey + Signature + Version
	size += 4*marshaller.Uint64Size + len(t.PublicKey) + len(t.Signature) + 1

	return size
}
