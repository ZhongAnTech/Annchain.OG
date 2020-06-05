package types

import (
	"github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/common"
)

type TxBaseType uint8

const (
	TxBaseTypeTx TxBaseType = iota
	TxBaseTypeSequencer
)

// Txi represents the basic structure that will exist on the DAG graph
// There may be many implementations of Txi so that the graph structure can support multiple tx types.
// Note: All fields not related to graph strcuture should be implemented in their own struct, not interface
type Txi interface {
	// Implemented by TxBase
	GetType() TxBaseType
	GetHeight() uint64
	SetHeight(uint64)
	GetWeight() uint64
	SetWeight(weight uint64)
	GetHash() types.Hash
	GetParents() types.Hashes // GetParents returns the common.Hash of txs that it directly proves.
	SetParents(hashes types.Hashes)
	String() string

	CalculateWeight(parents Txis) uint64
	Compare(tx Txi) bool      // Compare compares two txs, return true if they are the same.
	SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.
	GetNonce() uint64

	SetMineNonce(uint64)

	SetHash(h types.Hash)
	SetValid(b bool)
	Valid() bool

	// implemented by each tx type
	//GetBase() *TxBase
	Sender() common.Address
	//GetSender() *common.Address
	SetSender(addr common.Address)
	//Dump() string             // For logger dump

	//RawTxi() RawTxi // compressed txi

	//GetVersion() byte
	//IsVerified() VerifiedType
	//SetVerified(v VerifiedType)
}

type Hashable interface {
	CalcTxHash() types.Hash // TxHash returns a full tx common.Hash (parents sealed by PoW stage 2)
}

type Signable interface {
	SignatureTargets() []byte
}

type Dumpable interface {
	Dump() string
}
