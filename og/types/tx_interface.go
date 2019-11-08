package types

import (
	"github.com/annchain/OG/common"
)

type TxBaseType uint8

const (
	TxBaseTypeNormal TxBaseType = iota
	TxBaseTypeSequencer
)

// Txi represents the basic structure that will exist on the DAG graph
// There may be many implementations of Txi so that the graph structure can support multiple tx types.
// Note: All fields not related to graph strcuture should be implemented in their own struct, not interface
type Txi interface {
	// Implemented by TxBase
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() common.Hash
	//GetNonce() uint64
	Parents() common.Hashes // Parents returns the common.Hash of txs that it directly proves.
	//SetHash(h common.Hash)
	String() string
	//CalcTxHash() common.Hash    // TxHash returns a full tx common.Hash (parents sealed by PoW stage 2)
	//CalcMinedHash() common.Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	//SetInValid(b bool)
	//InValid() bool

	// implemented by each tx type
	//GetBase() *TxBase
	//Sender() common.Address
	//GetSender() *common.Address
	//SetSender(addr common.Address)
	//Dump() string             // For logger dump
	Compare(tx Txi) bool // Compare compares two txs, return true if they are the same.
	//SignatureTargets() []byte // SignatureTargets only returns the parts that needs to be signed by sender.

	//RawTxi() RawTxi // compressed txi

	// implemented by msgp
	//DecodeMsg(dc *msgp.Reader) (err error)
	//EncodeMsg(en *msgp.Writer) (err error)
	//MarshalMsg(b []byte) (o []byte, err error)
	//UnmarshalMsg(bts []byte) (o []byte, err error)
	//Msgsize() (s int)
	//GetVersion() byte
	//ToSmallCaseJson() ([]byte, error)
	//IsVerified() VerifiedType
	//SetVerified(v VerifiedType)
}

type Hashable interface {
	CalcTxHash() common.Hash // TxHash returns a full tx common.Hash (parents sealed by PoW stage 2)
}
type Signable interface {
	SignatureTargets() []byte
}

type Dumpable interface {
	Dump() string
}