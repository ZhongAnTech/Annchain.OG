package types

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/byteutil"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"strings"
)

type Tx struct {
	// graph structure info
	Hash        types.Hash
	ParentsHash types.Hashes
	MineNonce   uint64

	// tx info
	AccountNonce uint64
	From         common.Address
	To           common.Address
	Value        *math.BigInt
	TokenId      int32
	Data         []byte
	PublicKey    crypto.PublicKey
	Signature    crypto.Signature
	//confirm     time.Time
	// CanRecoverPubFromSig

	// Derived properties
	Height uint64
	Weight uint64

	valid bool
}

func (t *Tx) SetMineNonce(v uint64) {
	t.MineNonce = v
}

func (t *Tx) SetParents(hashes types.Hashes) {
	t.ParentsHash = hashes
}

func (t *Tx) SetWeight(weight uint64) {
	t.Weight = weight
}

func (t *Tx) SetValid(b bool) {
	t.valid = b
}

func (t *Tx) Valid() bool {
	return t.valid
}

func (t *Tx) SetSender(addr common.Address) {
	t.From = addr
}

func (t *Tx) SetHash(h types.Hash) {
	t.Hash = h
}

func (t *Tx) GetNonce() uint64 {
	return t.AccountNonce
}

func (t *Tx) Sender() common.Address {
	return t.From
}

func (t *Tx) SetHeight(height uint64) {
	t.Height = height
}

func (t *Tx) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash %s, pHash:[%s], from: %s, to: %s, value: %s,\n nonce: %d, signatute: %s, pubkey: %s, "+
		"minedNonce: %v, data: %x", t.Hash.Hex(),
		strings.Join(phashes, " ,"), t.From.Hex(), t.To.Hex(), t.Value,
		t.AccountNonce, t.Signature, hexutil.Encode(t.PublicKey.KeyBytes), t.MineNonce, t.Data)
}

func (t *Tx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	w := byteutil.NewBinaryWriter()

	w.Write(t.AccountNonce)
	w.Write(t.From.Bytes)
	w.Write(t.To.Bytes, t.Value.GetSigBytes(), t.Data, t.TokenId)
	return w.Bytes()
}

func (t *Tx) GetType() TxBaseType {
	return TxBaseTypeTx
}

func (t *Tx) GetHeight() uint64 {
	return t.Height
}

func (t *Tx) GetWeight() uint64 {
	if t.Weight == 0 {
		panic("implementation error: weight not initialized")
	}
	return t.Weight
}

func (t *Tx) GetHash() types.Hash {
	return t.Hash
}

func (t *Tx) GetParents() types.Hashes {
	return t.ParentsHash
}

func (t *Tx) String() string {
	return t.DebugString()
	//return fmt.Sprintf("T-%d-[%.10s]-%d", t.Height, t.GetHash().Hex(), t.Weight)
}

func (t *Tx) DebugString() string {
	s := t.GetHash().Hex()
	return fmt.Sprintf("Tx-H%d-W%d-N%d-[%s]", t.Height, t.Weight, t.AccountNonce, s[len(s)-8:])
}

//CalculateWeight  a core algorithm for tx sorting,
//a tx's weight must bigger than any of it's parent's weight  and bigger than any of it's elder transaction's
func (t *Tx) CalculateWeight(parents Txis) uint64 {
	var maxWeight uint64
	for _, p := range parents {
		if p.GetWeight() > maxWeight {
			maxWeight = p.GetWeight()
		}
	}
	return maxWeight + 1
}

func (t *Tx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Tx:
		if t.GetHash().Cmp(tx.GetHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}
