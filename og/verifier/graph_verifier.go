package verifier

import (
	types2 "github.com/annchain/OG/arefactor/og/types"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/ogcore/pool"

	"github.com/annchain/OG/status"
	"github.com/sirupsen/logrus"
)

// GraphVerifier verifies if the tx meets the OG hash and graph standards.
type GraphVerifier struct {
	Dag    og.IDag
	TxPool pool.ITxPool
	//Buffer *TxBuffer
}

func (c *GraphVerifier) String() string {
	return c.Name()
}
func (v *GraphVerifier) Independent() bool {
	return false
}

func (v *GraphVerifier) Name() string {
	return "GraphVerifier"
}

// getTxFromTempSource tries to get tx from anywhere but dag itself.
// return nil if not found in either txpool or buffer
func (v *GraphVerifier) getTxFromTempSource(hash types2.Hash) (txi types.Txi) {
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

func (v *GraphVerifier) getTxFromAnywhere(hash types2.Hash) (txi types.Txi, archived bool) {
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
	if currentTx.GetNonce() == 1 {
		ok = true
		return
	}
	seeked := map[types2.Hash]bool{}
	seekingHashes := types2.Hashes{}
	for _, parent := range currentTx.GetParents() {
		seekingHashes = append(seekingHashes, parent)
	}

	if status.ArchiveMode {
		if ptx := v.Dag.GetTxByNonce(currentTx.Sender(), currentTx.GetNonce()-1); ptx != nil {
			previousTx = ptx
			ok = true
			return
		}
		return nil, false
	}

	for len(seekingHashes) > 0 {
		head := seekingHashes[0]
		seekingHashes = seekingHashes[1:]

		txi, archived := v.getTxFromAnywhere(head)
		if txi != nil {
			// found. verify nonce
			if txi.GetType() != types.TxBaseTypeArchive {
				if txi.Sender() == currentTx.Sender() {
					// verify if the nonce is larger
					if txi.GetNonce() == currentTx.GetNonce()-1 {
						// good
						previousTx = txi
						ok = true
						return
					}
				}
			}

			switch txi.GetType() {
			case types.TxBaseTypeSequencer:
				// nothing to do, since all txs before seq should already be archived
			default:
				// may be somewhere else
				// enqueue header more if we are still in the temp area
				if archived {
					continue
				}
				for _, parent := range txi.GetParents() {
					if _, ok := seeked[parent]; !ok {
						seekingHashes = append(seekingHashes, parent)
						seeked[parent] = true
					}
				}
				continue

			}
		} else {
			// should not be here
			logrus.WithField("current tx ", currentTx).WithField("ancestor", head).Warn("ancestor not found: should not be here.")
			// this ancestor should already be in the dag. do nothing
		}
	}

	//// Here, the ancestor of the same From address must be in the dag.
	//// Once found, this tx must be the previous tx of currentTx because everyone behind sequencer is confirmed by sequencer
	//if ptx := v.Dag.GetTxByNonce(currentTx.Sender(), currentTx.GetNonce()-1); ptx != nil {
	//	previousTx = ptx
	//	ok = true
	//	return
	//}
	//return
	return
}

// get the nearest previous sequencer from txpool
func (v *GraphVerifier) getPreviousSequencer(currentSeq *types.Sequencer) (previousSeq *types.Sequencer, ok bool) {
	seeked := map[types2.Hash]bool{}
	seekingHashes := types2.Hashes{}
	// seekingHashes := list.New()
	seekingHashes = append(seekingHashes, currentSeq.GetHash())
	// seekingHashes.PushBack(currentSeq.GetHash())
	for len(seekingHashes) > 0 {
		head := seekingHashes[0]
		seekingHashes = seekingHashes[1:]
		// head := seekingHashes.Remove(seekingHashes.Front()).(common.Hash)
		txi, archived := v.getTxFromAnywhere(head)

		if txi != nil {
			switch txi.GetType() {
			case types.TxBaseTypeSequencer:
				// found seq, check nonce
				// verify if the nonce is larger
				if txi.(*types.Sequencer).Height == currentSeq.Height-1 {
					// good
					previousSeq = txi.(*types.Sequencer)
					ok = true
					return
				}
			default:
				break
			}
			// may be somewhere else
			// enqueue header more if we are still in the temp area
			if archived {
				continue
			}
			for _, parent := range txi.GetParents() {
				if _, ok := seeked[parent]; !ok {
					seekingHashes = append(seekingHashes, parent)
					// seekingHashes.PushBack(parent)
					seeked[parent] = true
				}
			}
		}
	}
	// Here, the ancestor of the same From address must be in the dag.
	// Once found, this tx must be the previous tx of currentTx because everyone behind sequencer is confirmed by sequencer
	if ptx := v.Dag.GetSequencerByHeight(currentSeq.Height - 1); ptx != nil {
		previousSeq = ptx
		ok = true
		return
	} else {
		logrus.Debug(ptx, currentSeq.Height-1)
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
	//if txi.IsVerified().IsGraphVerified() {
	//	return true
	//}
	ok = false
	if ok = v.verifyWeight(txi); !ok {
		logrus.WithField("tx", txi).Debug("tx failed on weight")
		return
	}
	if ok = v.verifyA3(txi); !ok {
		logrus.WithField("tx", txi).Debug("tx failed on graph A3")
		return
	}
	if ok = v.verifyB1(txi); !ok {
		logrus.WithField("tx", txi).Debug("tx failed on graph B1")
		return
	}
	if txi.GetType() == types.TxBaseTypeSequencer {
		seq := txi.(*types.Sequencer)
		if err := v.TxPool.IsBadSeq(seq); err != nil {
			logrus.WithField("seq ", seq).WithError(err).Warn("bad seq")
			return false
		}
	}

	txi.SetVerified(archive.VerifiedGraph)
	return true
}

func (v *GraphVerifier) verifyA3(txi types.Txi) bool {
	// constantly check the ancestors until the same one issued by me is found.
	// or nonce reaches 1

	if txi.GetType() == types.TxBaseTypeArchive {
		return true
	}
	if status.ArchiveMode {
		if txi.GetType() != types.TxBaseTypeSequencer {
			logrus.Warn("archive mode , only process archive")
			return false
		}
	}

	if txi.GetNonce() == 0 {
		return false
	}
	var ok bool

	// first tx check
	nonce, poolErr := v.TxPool.GetLatestNonce(txi.Sender())
	if txi.GetNonce() == 1 {
		// test claim: whether it should be 0

		if poolErr == nil {
			logrus.Debugf("nonce should not be 0. Latest nonce is %d and you should be larger than it", nonce)
		}
		// not found is good
		return poolErr != nil
	}
	dagNonce, _ := v.Dag.GetLatestNonce(txi.Sender())
	if poolErr != nil {
		//no related tx in txpool ,check dag
		if dagNonce != txi.GetNonce()-1 {
			logrus.WithField("current nonce ", txi.GetNonce()).WithField("dag nonce ", dagNonce).WithField("tx", txi).Debug("previous tx  not found for address")
			// fail if not good
			return false
		}
		goto Out
	}
	if nonce < txi.GetNonce()-1 {
		logrus.WithField("current nonce ", txi.GetNonce()).WithField("pool nonce ", nonce).WithField("tx", txi).Debug("previous tx  not found for address")
		// fail if not good
		return false
	}
	// check txpool queue first
	if dagNonce != nonce {
		_, ok = v.getMyPreviousTx(txi)
		if !ok {
			logrus.WithField("tx", txi).Debug("previous tx not found")
			// fail if not good
			return ok
		}
	}

Out:

	switch txi.GetType() {
	// no additional check
	case types.TxBaseTypeSequencer:
		seq := txi.(*types.Sequencer)
		// to check if there is a lower seq height in the path behind
		_, ok = v.getPreviousSequencer(seq)
		if !ok {
			logrus.WithField("tx", txi).Debug("previous seq not found")
		}

		return ok
	default:
	}
	return true
}

func (v *GraphVerifier) verifyB1(txi types.Txi) bool {
	// compare the sequencer id
	return true
}

func (v *GraphVerifier) verifyWeight(txi types.Txi) bool {
	var parents types.Txis
	for _, pHash := range txi.GetParents() {
		parent := v.TxPool.Get(pHash)
		if parent == nil {
			parent = v.Dag.GetTx(pHash)
		}
		if parent == nil {
			logrus.WithField("parent hash ", pHash).Warn("parent not found")
			return false
		}
		parents = append(parents, parent)
	}
	return txi.CalculateWeight(parents) == txi.GetWeight()
}
