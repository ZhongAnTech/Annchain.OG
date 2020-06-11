package verifier

import (
	"fmt"
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestConsensusVerifier_Verify(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	tx := archive.RandomTx()
	//fmt.Println(tx)
	pub, priv := og_interface.Signer.RandomKeyPair()
	tx.From = nil
	fmt.Println(tx.SignatureTargets())
	tx.Signature = og_interface.Signer.Sign(priv, tx.SignatureTargets()).SignatureBytes
	tx.SetHash(tx.CalcTxHash())
	tx.From = nil
	//fmt.Println(tx,"hehe")
	v := verifier.TxFormatVerifier{NoVerifyMindHash: true, NoVerifyMaxTxHash: true}
	ok := v.Verify(tx)
	fmt.Println(tx, "hihi")
	if !ok {
		t.Fatal(ok)
	}
	if *tx.From != pub.Address() {
		t.Fatal(tx.From, pub.Address())
	}
	fmt.Println(tx.From, pub.Address())

}
