package og

import (
	"container/list"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

// GraphVerifier verifies if the tx meets the standards
type Verifier interface {
	Verify(t types.Txi) bool
	Name() string
}

type TxFormatVerifier struct {
	Signer       crypto.Signer
	CryptoType   crypto.CryptoType
	MaxTxHash    types.Hash // The difficultiy of TxHash
	MaxMinedHash types.Hash // The difficultiy of MinedHash
}

func (v *TxFormatVerifier) Name() string {
	return "TxFormatVerifier"
}

func (v *TxFormatVerifier) Verify(t types.Txi) bool {
	if !v.VerifyHash(t) {
		logrus.WithField("tx", t).Debug("Hash not valid")
		return false
	}
	if !v.VerifySignature(t) {
		logrus.WithField("tx", t).Debug("Signature not valid")
		return false
	}
	if !v.VerifySourceAddress(t) {
		logrus.WithField("tx", t).Debug("Source address not valid")
		return false
	}
	return true
}

func (v *TxFormatVerifier) VerifyHash(t types.Txi) bool {
	if !(t.CalcMinedHash().Cmp(v.MaxMinedHash) < 0) {
		logrus.WithField("tx", t).WithField("hash", t.CalcMinedHash().String()).Debug("MinedHash is not less than MaxMinedHash")
		return false
	}
	if t.CalcTxHash() != t.GetTxHash() {
		logrus.WithField("tx", t).WithField("hash", t.CalcMinedHash().String()).Debug("TxHash is not aligned with content")
		return false
	}
	if !(t.GetTxHash().Cmp(v.MaxTxHash) < 0) {
		logrus.WithField("tx", t).WithField("hash", t.CalcMinedHash().String()).Debug("TxHash is not less than MaxTxHash")
		return false
	}
	return true
}

func (v *TxFormatVerifier) VerifySignature(t types.Txi) bool {
	base := t.GetBase()
	return v.Signer.Verify(
		crypto.PublicKey{Type: v.CryptoType, Bytes: base.PublicKey},
		crypto.Signature{Type: v.CryptoType, Bytes: base.Signature},
		t.SignatureTargets())
}

func (v *TxFormatVerifier) VerifySourceAddress(t types.Txi) bool {
	switch t.(type) {
	case *types.Tx:
		return t.(*types.Tx).From.Bytes == v.Signer.Address(crypto.PublicKeyFromBytes(v.CryptoType, t.GetBase().PublicKey)).Bytes
	case *types.Sequencer:
		return t.(*types.Sequencer).Issuer.Bytes == v.Signer.Address(crypto.PublicKeyFromBytes(v.CryptoType, t.GetBase().PublicKey)).Bytes
	default:
		return true
	}
}

// GraphVerifier verifies if the tx meets the OG hash and graph standards.
type GraphVerifier struct {
	Dag    IDag
	TxPool ITxPool
	//Buffer *TxBuffer
}

func (v *GraphVerifier) Name() string {
	return "GraphVerifier"
}

// getTxFromTempSource tries to get tx from anywhere but dag itself.
// return nil if not found in either txpool or buffer
func (v *GraphVerifier) getTxFromTempSource(hash types.Hash) (txi types.Txi) {
	// Re-think. Do we really need to check buffer?

	//if v.Buffer != nil {
	//	txi = v.Buffer.GetFromBuffer(hash)
	//	if txi != nil {
	//		return
	//	}
	//}

	if v.TxPool != nil {
		txi = v.TxPool.Get(hash)
		if txi != nil {
			return
		}
	}
	return
}

func (v *GraphVerifier) getTxFromAnywhere(hash types.Hash) (txi types.Txi, archived bool) {
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
// return true if found, or false if not found in txpool or in dag
func (v *GraphVerifier) getMyPreviousTx(currentTx types.Txi) (previousTx types.Txi, ok bool) {
	if currentTx.GetNonce() == 0 {
		ok = true
		return
	}
	seeked := map[types.Hash]bool{}
	seekingHashes := list.New()
	for _, parent := range currentTx.Parents() {
		seekingHashes.PushBack(parent)
	}

	for seekingHashes.Len() > 0 {
		head := seekingHashes.Remove(seekingHashes.Front()).(types.Hash)
		//logrus.WithField("ancestor", head).WithField("tx", currentTx).Debug("fetching ancestor tx")

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
			// should not be here
			logrus.WithField("current tx ", currentTx).WithField("ancestor", head).Warn("ancestor not found: should not be here.")
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
func (v *GraphVerifier) getPreviousSequencer(currentSeq *types.Sequencer) (previousSeq *types.Sequencer, ok bool) {
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

// Verify do the graph validation according to the following rules that are marked as [My job].
// Graph standards:
// A1: [My job] Randomly choose 2 tips.
// A2: [Not used] Node's parent cannot be its grandparents or ancestors.
// A3: [My job] Nodes produced by same source must be sequential (tx nonce ++).
// A4: [------] Double spending once A3 is not followed, whatever there is actual double spending.
// A5: [Pool's job] If A3 is followed but there is still double spending (tx nonce collision), keep the forked tx with smaller hash
// A6: [My job] Node cannot reference two un-ordered nodes as its parents
// B1: [My job] Nodes that are confirmed by at least N (=2) sequencers cannot be referenced.
// B2: [My job] Two layer hash validation
// Basically Verify checks whether txs are in their nonce order
func (v *GraphVerifier) Verify(txi types.Txi) (ok bool) {
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

func (v *GraphVerifier) verifyA3(txi types.Txi) bool {
	// constantly check the ancestors until the same one issued by me is found.
	// or nonce reaches 0

	// zero check
	if txi.GetNonce() == 0 {
		// test claim: whether it should be 0
		v, err := v.TxPool.GetLatestNonce(txi.Sender())
		if err == nil {
			logrus.Debugf("nonce shoud not be 0. Latest nonce is %d and you should be larger than it", v)
		}
		// not found is good
		return err != nil
	}
	// check txpool queue first
	_, ok := v.getMyPreviousTx(txi)
	if !ok {
		logrus.WithField("tx", txi).Debug("previous tx not found")
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

func (v *GraphVerifier) verifyB1(txi types.Txi) bool {
	// compare the sequencer id
	return true
}
