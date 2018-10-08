package og

import (
	"container/list"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

// Verifier verifies if the tx meets the hash and graph standards.
type Verifier struct {
	Signer       crypto.Signer
	CryptoType   crypto.CryptoType
	Dag          IDag
	TxPool       ITxPool
	Buffer       *TxBuffer
	MaxTxHash    types.Hash // The difficultiy of TxHash
	MaxMinedHash types.Hash // The difficultiy of MinedHash
}

func (v *Verifier) VerifyHash(t types.Txi) bool {
	return (t.CalcMinedHash().Cmp(v.MaxMinedHash) < 0) &&
		t.CalcTxHash() == t.GetTxHash() &&
		(t.GetTxHash().Cmp(v.MaxTxHash) < 0)

}

func (v *Verifier) VerifySignature(t types.Txi) bool {
	base := t.GetBase()
	return v.Signer.Verify(
		crypto.PublicKey{Type: v.CryptoType, Bytes: base.PublicKey},
		crypto.Signature{Type: v.CryptoType, Bytes: base.Signature},
		t.SignatureTargets())
}

func (v *Verifier) VerifySourceAddress(t types.Txi) bool {
	switch t.(type) {
	case *types.Tx:
		return t.(*types.Tx).From.Bytes == v.Signer.Address(crypto.PublicKeyFromBytes(v.CryptoType, t.GetBase().PublicKey)).Bytes
	case *types.Sequencer:
		return t.(*types.Sequencer).Issuer.Bytes == v.Signer.Address(crypto.PublicKeyFromBytes(v.CryptoType, t.GetBase().PublicKey)).Bytes
	default:
		return true
	}
}

// getTxFromTempSource tries to get tx from anywhere but dag itself.
// return nil if not found in either txpool or buffer
func (v *Verifier) getTxFromTempSource(hash types.Hash) (txi types.Txi) {
	if v.Buffer != nil {
		txi = v.Buffer.GetFromBuffer(hash)
		if txi != nil {
			return
		}
	}

	if v.TxPool != nil {
		txi = v.TxPool.Get(hash)
		if txi != nil {
			return
		}
	}
	return
}

func (v *Verifier) getTxFromAnywhere(hash types.Hash) (txi types.Txi, archived bool) {
	txi = v.getTxFromTempSource(hash)
	if txi != nil {
		archived = false
		return
	}
	if v.Dag != nil {
		txi = v.Dag.GetTx(hash)
		if txi != nil {
			archived = true
			return
		}
	}
	return
}

// getMyPreviousTx tries to fetch the tx that is announced by the same source with nonce = current nonce -1
// return true if found, or false if not found in txpool (then it maybe in dag)
func (v *Verifier) getMyPreviousTx(currentTx types.Txi) (previousTx types.Txi, ok bool) {
	if currentTx.GetNonce() == 0 {
		ok = true
		return
	}
	seeked := map[types.Hash]bool{}
	seekingHashes := list.New()
	for _, parent := range currentTx.Parents(){
		seekingHashes.PushBack(parent)
	}

	for seekingHashes.Len() > 0 {
		head := seekingHashes.Remove(seekingHashes.Front()).(types.Hash)
		logrus.WithField("hash", head).Debug("fetching previous tx")

		txi, archived := v.getTxFromAnywhere(head)
		if txi != nil {
			// found. verify nonce
			if txi.Sender() == currentTx.Sender() {
				// verify if the nonce is larger
				if txi.GetNonce() == currentTx.GetNonce()-1 {
					// good
					previousTx = txi
					ok = true
					return
				}
			}

			switch txi.GetType() {
			case types.TxBaseTypeNormal:
				// may be somewhere else
				// enqueue header more if we are still in the temp area
				if archived {
					continue
				}
				for _, parent := range txi.Parents() {
					if _, ok := seeked[parent]; !ok {
						seekingHashes.PushBack(parent)
						seeked[parent] = true
					}
				}
				continue
			case types.TxBaseTypeSequencer:
				// nothing to do, since all txs before seq should already be archived
			}
		} else {
			// this ancestor should already be in the dag. do nothing
		}
	}
	// Here, the ancestor of the same From address must be in the dag.
	// Once found, this tx must be the previous tx of currentTx because everyone behind sequencer is confirmed by sequencer
	if ptx := v.Dag.GetTxByNonce(currentTx.Sender(), currentTx.GetNonce()-1); ptx != nil {
		previousTx = ptx
		ok = true
		return
	}
	return
}

// get the nearest previous sequencer from txpool
func (v *Verifier) getPreviousSequencer(currentSeq *types.Sequencer) (previousSeq *types.Sequencer, ok bool) {
	seeked := map[types.Hash]bool{}
	seekingHashes := list.New()
	seekingHashes.PushBack(currentSeq.GetTxHash())
	for seekingHashes.Len() > 0 {
		head := seekingHashes.Remove(seekingHashes.Front()).(types.Hash)
		txi, archived := v.getTxFromAnywhere(head)

		if txi != nil {
			switch txi.GetType() {
			case types.TxBaseTypeNormal:
				break
			case types.TxBaseTypeSequencer:
				// found seq, check nonce
				// verify if the nonce is larger
				if txi.(*types.Sequencer).Id == currentSeq.Id-1 {
					// good
					previousSeq = txi.(*types.Sequencer)
					ok = true
					return
				}
			}
			// may be somewhere else
			// enqueue header more if we are still in the temp area
			if archived {
				continue
			}
			for _, parent := range txi.Parents() {
				if _, ok := seeked[parent]; !ok {
					seekingHashes.PushBack(parent)
					seeked[parent] = true
				}
			}
		}
	}
	// Here, the ancestor of the same From address must be in the dag.
	// Once found, this tx must be the previous tx of currentTx because everyone behind sequencer is confirmed by sequencer
	if ptx := v.Dag.GetSequencerById(currentSeq.Id - 1); ptx != nil {
		previousSeq = ptx
		ok = true
		return
	}

	return
}

// VerifyGraphOrder do the graph validation according to the following rules that are marked as [My job].
// Graph standards:
// A1: [My job] Randomly choose 2 tips.
// A2: [Not used] Node's parent cannot be its grandparents or ancestors.
// A3: [My job] Nodes produced by same source must be sequential (tx nonce ++).
// A4: [------] Double spending once A3 is not followed, whatever there is actual double spending.
// A5: [Pool's job] If A3 is followed but there is still double spending (tx nonce collision), keep the forked tx with smaller hash
// A6: [My job] Node cannot reference two un-ordered nodes as its parents
// B1: [My job] Nodes that are confirmed by at least N (=2) sequencers cannot be referenced.
// B2: [My job] Two layer hash validation
// Basically VerifyGraphOrder checks whether txs are in their nonce order
func (v *Verifier) VerifyGraphOrder(txi types.Txi) (ok bool) {
	ok = false
	if ok = v.verifyA3(txi); !ok {
		logrus.WithField("tx", txi).Debug("tx failed on graph A3")
		return
	}
	if ok = v.verifyB1(txi); !ok {
		logrus.WithField("tx", txi).Debug("tx failed on graph B1")
		return
	}
	return true
}

func (v *Verifier) verifyA3(txi types.Txi) bool {
	// constantly check the ancestors until the same one issued by me is found.
	// or nonce reaches 0

	// zero check
	if txi.GetNonce() == 0 {
		// test claim: whether it should be 0
		_, err := v.TxPool.GetLatestNonce(txi.Sender())
		// not found is good
		return err != nil
	}
	// check txpool queue first
	_, ok := v.getMyPreviousTx(txi)
	if !ok {
		// fail if not good
		return ok
	}

	switch txi.GetType() {
	case types.TxBaseTypeNormal:
		// no additional check
	case types.TxBaseTypeSequencer:
		seq := txi.(*types.Sequencer)
		// to check if there is a lower seq id in the path behind
		_, ok := v.getPreviousSequencer(seq)
		return ok
	}
	return true
}

func (v *Verifier) verifyB1(txi types.Txi) bool {
	// compare the sequencer id
	return true
}
