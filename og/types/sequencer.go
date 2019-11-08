package types

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
	"golang.org/x/crypto/sha3"
)

type Sequencer struct {
	// graph structure info
	Hash        common.Hash
	ParentsHash common.Hashes
	Height      uint64
	MineNonce   uint64
	Weight      uint64

	AccountNonce   uint64
	Issuer         *common.Address
	BlsJointSig    hexutil.Bytes
	BlsJointPubKey hexutil.Bytes
	StateRoot      common.Hash
	Proposing      bool `msg:"-"` // is the sequencer is proposal ,did't commit yet ,use this flag to avoid bls sig verification failed
}

func (s Sequencer) CalcTxHash() (hash common.Hash) {
	// TODO: double check the hash content
	w := types.NewBinaryWriter()

	for _, ancestor := range s.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	w.Write(s.Weight)
	w.Write(s.BlsJointSig)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (s *Sequencer) SignatureTargets() []byte {
	w := types.NewBinaryWriter()

	w.Write(s.BlsJointPubKey, s.AccountNonce)
	w.Write(s.Issuer.Bytes)

	w.Write(s.Height, s.Weight, s.StateRoot.Bytes)
	for _, parent := range s.Parents() {
		w.Write(parent.Bytes)
	}
	return w.Bytes()
}

func (s Sequencer) GetType() TxBaseType {
	return TxBaseTypeSequencer
}

func (s Sequencer) GetHeight() uint64 {
	return s.Height
}

func (s Sequencer) GetWeight() uint64 {
	if s.Weight == 0 {
		panic("implementation error: weight not initialized")
	}
	return t.Weight
}

func (s Sequencer) GetTxHash() common.Hash {
	if s.Hash.Empty() {
		s.CalcTxHash()
	}
	return s.Hash
}

func (s Sequencer) Parents() common.Hashes {
	return s.ParentsHash
}

func (s Sequencer) String() string {
	if s.Issuer == nil {
		return fmt.Sprintf("Sq-[nil]-%d", s.AccountNonce)
	} else {
		return fmt.Sprintf("Sq-[%.10s]-%d", s.Issuer.String(), s.AccountNonce)
	}
}

func (s Sequencer) CalculateWeight(parents Txis) uint64 {
	var maxWeight uint64
	for _, p := range parents {
		if p.GetWeight() > maxWeight {
			maxWeight = p.GetWeight()
		}
	}
	return maxWeight + 1
}

func (s Sequencer) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *Sequencer:
		if s.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}
