// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package og

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/status"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"math/big"
)

// GraphVerifier verifies if the tx meets the standards
type Verifier interface {
	Verify(t types.Txi) bool
	Name() string
	String() string
	Independent() bool
}

type TxFormatVerifier struct {
	MaxTxHash         common.Hash // The difficulty of TxHash
	MaxMinedHash      common.Hash // The difficulty of MinedHash
	NoVerifyMindHash  bool
	NoVerifyMaxTxHash bool
}

//consensus related verification
type ConsensusVerifier struct {
	VerifyCampaign   func(cp *tx_types.Campaign) bool
	VerifyTermChange func(cp *tx_types.TermChange) bool
	VerifySequencer  func(cp *tx_types.Sequencer) bool
}

func (c *ConsensusVerifier) Verify(t types.Txi) bool {
	switch tx := t.(type) {
	case *tx_types.Tx:
		return true
	case *tx_types.Archive:
		return true
	case *tx_types.ActionTx:
		return true
	case *tx_types.Sequencer:
		return c.VerifySequencer(tx)
	case *tx_types.Campaign:
		return c.VerifyCampaign(tx)
	case *tx_types.TermChange:
		return c.VerifyTermChange(tx)
	default:
		return false
	}
	return false
}

func (v *TxFormatVerifier) Name() string {
	return "TxFormatVerifier"
}

func (v *TxFormatVerifier) Independent() bool {
	return true
}

func (c *ConsensusVerifier) Name() string {
	return "ConsensusVerifier"
}

func (v *ConsensusVerifier) Independent() bool {
	return false
}

func (c *TxFormatVerifier) String() string {
	return c.Name()
}

func (c *GraphVerifier) String() string {
	return c.Name()
}
func (v *GraphVerifier) Independent() bool {
	return false
}

func (c *ConsensusVerifier) String() string {
	return c.Name()
}

func (v *TxFormatVerifier) Verify(t types.Txi) bool {
	if t.IsVerified().IsFormatVerified() {
		return true
	}
	if !v.VerifyHash(t) {
		logrus.WithField("tx", t).Debug("Hash not valid")
		return false
	}
	if !v.VerifySignature(t) {
		logrus.WithField("sig targets ", hex.EncodeToString(t.SignatureTargets())).WithField("tx dump: ", t.Dump()).WithField("tx", t).Debug("Signature not valid")
		return false
	}
	t.SetVerified(types.VerifiedFormat)
	return true
}

func (v *TxFormatVerifier) VerifyHash(t types.Txi) bool {
	if !v.NoVerifyMindHash {
		calMinedHash := t.CalcMinedHash()
		if !(calMinedHash.Cmp(v.MaxMinedHash) < 0) {
			logrus.WithField("tx", t).WithField("hash", calMinedHash).Debug("MinedHash is not less than MaxMinedHash")
			return false
		}
	}
	if calcHash := t.CalcTxHash(); calcHash != t.GetTxHash() {
		logrus.WithField("calcHash ", calcHash).WithField("tx", t).WithField("hash", t.GetTxHash()).Debug("TxHash is not aligned with content")
		return false
	}

	if !v.NoVerifyMaxTxHash && !(t.GetTxHash().Cmp(v.MaxTxHash) < 0) {
		logrus.WithField("tx", t).WithField("hash", t.GetTxHash()).Debug("TxHash is not less than MaxTxHash")
		return false
	}
	return true
}

func (v *TxFormatVerifier) VerifySignature(t types.Txi) bool {
	if t.GetType() == types.TxBaseTypeArchive {
		return true
	}
	base := t.GetBase()

	if !crypto.Signer.CanRecoverPubFromSig() {
		if t.GetSender() == nil {
			logrus.Warn("verify sig failed, from is nil")
			return false
		}
		ok := crypto.Signer.Verify(
			crypto.Signer.PublicKeyFromBytes(base.PublicKey),
			crypto.Signature{Type: crypto.Signer.GetCryptoType(), Bytes: base.Signature},
			t.SignatureTargets())
		return ok
	}

	R, S, Vb, err := v.SignatureValues(base.Signature)
	if err != nil {
		logrus.WithError(err).Debug("verify sig failed")
		return false
	}
	if Vb.BitLen() > 8 {
		logrus.WithError(err).Debug("v len error")
		return false
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, false) {
		logrus.WithError(err).Debug("v len error")
		return false
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	sighash := Sha256(t.SignatureTargets())
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		logrus.WithError(err).Debug("sig verify failed")
	}
	if len(pub) == 0 || pub[0] != 4 {
		err := errors.New("invalid public key")
		logrus.WithError(err).Debug("verify sig failed")
	}
	var addr common.Address
	copy(addr.Bytes[:], crypto.Keccak256(pub[1:])[12:])
	t.SetSender(addr)
	return true
}

func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (t *TxFormatVerifier) SignatureValues(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		return r, s, v, fmt.Errorf("wrong size for signature: got %d, want 65", len(sig))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func (v *TxFormatVerifier) VerifySourceAddress(t types.Txi) bool {
	if crypto.Signer.CanRecoverPubFromSig() {
		//address was set by recovering signature ,
		return true
	}
	switch t.(type) {
	case *tx_types.Tx:
		return t.(*tx_types.Tx).From.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
	case *tx_types.Sequencer:
		return t.(*tx_types.Sequencer).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
	case *tx_types.Campaign:
		return t.(*tx_types.Campaign).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
	case *tx_types.TermChange:
		return t.(*tx_types.TermChange).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
	case *tx_types.Archive:
		return true
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
func (v *GraphVerifier) getTxFromTempSource(hash common.Hash) (txi types.Txi) {
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

func (v *GraphVerifier) getTxFromAnywhere(hash common.Hash) (txi types.Txi, archived bool) {
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
	seeked := map[common.Hash]bool{}
	seekingHashes := common.Hashes{}
	for _, parent := range currentTx.Parents() {
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
				for _, parent := range txi.Parents() {
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
func (v *GraphVerifier) getPreviousSequencer(currentSeq *tx_types.Sequencer) (previousSeq *tx_types.Sequencer, ok bool) {
	seeked := map[common.Hash]bool{}
	seekingHashes := common.Hashes{}
	// seekingHashes := list.New()
	seekingHashes = append(seekingHashes, currentSeq.GetTxHash())
	// seekingHashes.PushBack(currentSeq.GetTxHash())
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
				if txi.(*tx_types.Sequencer).Height == currentSeq.Height-1 {
					// good
					previousSeq = txi.(*tx_types.Sequencer)
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
			for _, parent := range txi.Parents() {
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
	if txi.IsVerified().IsGraphVerified() {
		return true
	}
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
	txi.SetVerified(types.VerifiedGraph)
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
			logrus.WithField("current nonce ", txi.GetNonce()-1).WithField("dag nonce ", dagNonce).WithField("tx", txi).Debug("previous tx  not found for address")
			// fail if not good
			return false
		}
		goto Out
	}
	if nonce != txi.GetNonce()-1 {
		logrus.WithField("current nonce ", txi.GetNonce()-1).WithField("pool nonce ", nonce).WithField("tx", txi).Debug("previous tx  not found for address")
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
		seq := txi.(*tx_types.Sequencer)
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
	for _, pHash := range txi.Parents() {
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
