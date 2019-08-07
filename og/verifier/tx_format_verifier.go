package verifier

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"math/big"
)

type TxFormatVerifier struct {
	MaxTxHash         common.Hash // The difficulty of TxHash
	MaxMinedHash      common.Hash // The difficulty of MinedHash
	NoVerifyMindHash  bool
	NoVerifyMaxTxHash bool
	NoVerifySignatrue bool
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
	if t.IsVerified().IsFormatVerified() {
		return true
	}
	if !v.VerifyHash(t) {
		logrus.WithField("tx", t).Debug("Hash not valid")
		return false
	}
	if v.NoVerifySignatrue {
		if !v.VerifySignature(t) {
			logrus.WithField("sig targets ", hex.EncodeToString(t.SignatureTargets())).WithField("tx dump: ", t.Dump()).WithField("tx", t).Debug("Signature not valid")
			return false
		}
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
