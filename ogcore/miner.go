package ogcore

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/byteutil"
	"github.com/annchain/OG/og/types"
	"golang.org/x/crypto/sha3"
)

type Miner struct {
}

func (m *Miner) CalcMinedHashTx(tx types.Tx) (hash common.Hash) {
	w := byteutil.NewBinaryWriter()
	//if !CanRecoverPubFromSig {
	w.Write(tx.PublicKey.ToBytes())
	//}
	w.Write(tx.Signature.ToBytes(), tx.MineNonce)
	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}

func (m *Miner) CalcMinedHashSeq(seq types.Sequencer) (hash common.Hash) {
	w := byteutil.NewBinaryWriter()

	for _, ancestor := range seq.ParentsHash {
		w.Write(ancestor.Bytes)
	}
	// do not use Height to calculate tx hash.
	//w.Write(s.Weight)
	w.Write(seq.Signature)

	result := sha3.Sum256(w.Bytes())
	hash.MustSetBytes(result[0:], common.PaddingNone)
	return
}
