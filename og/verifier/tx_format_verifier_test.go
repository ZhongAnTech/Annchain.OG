package verifier

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"

	"testing"
	"time"
)

type TestSigner struct {
	crypto.SignerSecp256k1
}

func (s *TestSigner) CanRecoverPubFromSig() bool {
	return true
}

func TestVerify(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	pub, priv := signer.RandomKeyPair()
	var txis types.Txis
	//var sigTerGets [][]byte
	addr := pub.Address()
	for i := 0; i < 10000; i++ {
		tx := archive.RandomTx()
		tx.From = &addr
		tx.Signature = signer.Sign(priv, tx.SignatureTargets()).SignatureBytes
		tx.PublicKey = pub.KeyBytes
		txis = append(txis, tx)
	}
	v := TxFormatVerifier{NoVerifyMineHash: true, NoVerifyMaxTxHash: true}
	now := time.Now()
	fmt.Println("start ", now)
	for i, tx := range txis {
		ok := v.VerifySignature(tx)
		if !ok {
			t.Fatal(ok, tx, i)
		}
	}
	fmt.Println("used ", time.Since(now))
	start := time.Now()
	newSigner := &TestSigner{}
	crypto.Signer = newSigner
	fmt.Println(crypto.Signer.CanRecoverPubFromSig())
	for i, tx := range txis {
		ok := v.VerifySignature(tx)
		if !ok {
			t.Fatal(ok, tx, i)
		}
	}
	fmt.Println("used ", time.Since(start))

}
