package verifier

import (
	"fmt"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestConsensusVerifier_Verify(t *testing.T) {
	logrus.SetLevel(logrus.TraceLevel)
	tx := tx_types.RandomTx()
	//fmt.Println(tx)
	pub, priv := crypto.Signer.RandomKeyPair()
	tx.From = nil
	fmt.Println(tx.SignatureTargets())
	tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes
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
