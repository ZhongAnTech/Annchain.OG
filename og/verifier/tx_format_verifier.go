package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/miner"
	"github.com/annchain/OG/og/types"

	"github.com/sirupsen/logrus"
	"math/big"
)

type TxFormatVerifier struct {
	MaxTxHash         common.Hash // The difficulty of TxHash
	MaxMinedHash      common.Hash // The difficulty of MinedHash
	NoVerifyHash      bool
	NoVerifySignatrue bool
	powMiner          miner.PoWMiner
}

func (v *TxFormatVerifier) Name() string {
	return "TxFormatVerifier"
}

func (c *TxFormatVerifier) String() string {
	return c.Name()
}

func (v *TxFormatVerifier) Independent() bool {
	return true
}

func (v *TxFormatVerifier) Verify(t types.Txi) bool {
	// NOTE: disabled shortcut verification because of complexity.
	//if t.IsVerified().IsFormatVerified() {
	//	return true
	//}
	if !v.NoVerifyHash && !v.VerifyHash(t) {
		logrus.WithField("tx", t).Debug("Hash not valid")
		return false
	}
	if !v.NoVerifySignatrue && !v.VerifySignature(t) {
		logrus.WithField("sig targets ", hex.EncodeToString(t.SignatureTargets())).WithField("tx", t).Warn("signature not valid")
		return false
	}
	return true
}

func (v *TxFormatVerifier) VerifyHash(t types.Txi) bool {
	txType := t.GetType()

	switch txType {
	case types.TxBaseTypeTx:
		// hash under pow
		if !v.powMiner.IsGoodTx(t.(*types.Tx), v.MaxMinedHash, v.MaxTxHash) {
			logrus.WithField("tx", t).Debug("Hash is not under pow limit")
			return false
		}
	case types.TxBaseTypeSequencer:
		// hash under pow
		if !v.powMiner.IsGoodSequencer(t.(*types.Sequencer), v.MaxMinedHash, v.MaxTxHash) {
			logrus.WithField("tx", t).Debug("Hash is not under pow limit")
			return false
		}
	}
	return true
}

func (v *TxFormatVerifier) VerifyFrom(t types.Txi) bool {
	//if t.GetSender() == nil {
	//	logrus.Warn("verify sig failed, from is nil")
	//	return false
	//}
	if crypto.Signer.CanRecoverPubFromSig() {

	}
	return true
}

func (v *TxFormatVerifier) VerifySignature(t types.Txi) bool {
	//if t.GetType() == archive2.TxBaseTypeArchive {
	//	return true
	//}
	txType := t.GetType()

	switch txType {
	case types.TxBaseTypeTx:
		tx := t.(*types.Tx)
		ok := crypto.Signer.Verify(tx.PublicKey, tx.Signature, t.SignatureTargets())
		if crypto.Signer.CanRecoverPubFromSig() {
			pub, err := crypto.PublicKeyFromSignature(tx.Hash, &tx.Signature)
			if err != nil {
				logrus.WithError(err).Warn("error on recovering pubkey from sig")
				return false
			}
			var addr common.Address
			copy(addr.Bytes[:], crypto.Keccak256(pub.KeyBytes)[12:])
			t.SetSender(addr)
		}
		return ok
	case types.TxBaseTypeSequencer:
		//tx := t.(*types.Sequencer)
		// TODO: what should happen here?
		//ok := crypto.Signer.Verify(tx.PublicKey, tx.Signature, t.SignatureTargets())
		return true
		//return ok
	}
	return false
}

//func (v *TxFormatVerifier) VerifySourceAddress(t types.Txi) bool {
//	if crypto.Signer.CanRecoverPubFromSig() {
//		//address was set by recovering signature ,
//		return true
//	}
//	switch t.(type) {
//	case *archive2.Tx:
//		return t.(*archive2.Tx).From.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
//	case *types.Sequencer:
//		return t.(*types.Sequencer).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
//	case *campaign.Campaign:
//		return t.(*campaign.Campaign).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
//	case *campaign.TermChange:
//		return t.(*campaign.TermChange).Issuer.Bytes == crypto.Signer.Address(crypto.Signer.PublicKeyFromBytes(t.GetBase().PublicKey)).Bytes
//	case *archive.Archive:
//		return true
//	default:
//		return true
//	}
//}

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

func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}
