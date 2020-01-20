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
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/magiconair/properties/assert"
	"github.com/sirupsen/logrus"
	"testing"
)

func buildTx(from common.Address, accountNonce uint64) *tx_types.Tx {
	tx := tx_types.RandomTx()
	tx.AccountNonce = accountNonce
	tx.From = &from
	return tx
}

func buildSeq(from common.Address, accountNonce uint64, id uint64) *tx_types.Sequencer {
	tx := tx_types.RandomSequencer()
	tx.AccountNonce = accountNonce
	tx.Issuer = &from
	return tx
}

func setParents(tx types.Txi, parents []types.Txi) {
	tx.GetBase().ParentsHash = common.Hashes{}
	for _, parent := range parents {
		tx.GetBase().ParentsHash = append(tx.GetBase().ParentsHash, parent.GetTxHash())
	}
}

func judge(t *testing.T, actual interface{}, want interface{}, id int) {
	assert.Equal(t, actual, want)
	if actual != want {
		logrus.Warnf("Assert %d, expected %t, actual %t.", id, want, actual)
	}
}

// A3: Nodes produced by same source must be sequential (tx nonce ++).
func TestA3(t *testing.T) {
	pool := &dummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := common.HexToAddress("0x0001")
	addr2 := common.HexToAddress("0x0002")

	txs := []types.Txi{
		buildSeq(addr1, 1, 1),
		buildTx(addr1, 2),
		buildTx(addr2, 0),
		buildTx(addr1, 3),
		buildTx(addr2, 1),

		buildTx(addr2, 2),
		buildTx(addr1, 4),
		buildTx(addr2, 3),
		buildTx(addr1, 2),
		buildTx(addr1, 4),

		buildTx(addr2, 4),
		buildTx(addr1, 3),
	}

	setParents(txs[0], []types.Txi{})
	setParents(txs[1], []types.Txi{txs[0]})
	setParents(txs[2], []types.Txi{txs[1], txs[7]})
	setParents(txs[3], []types.Txi{txs[2], txs[8]})
	setParents(txs[4], []types.Txi{txs[3], txs[9]})

	setParents(txs[5], []types.Txi{txs[4], txs[10]})
	setParents(txs[6], []types.Txi{txs[5]})
	setParents(txs[7], []types.Txi{txs[0]})
	setParents(txs[8], []types.Txi{txs[7]})
	setParents(txs[9], []types.Txi{txs[8]})

	setParents(txs[10], []types.Txi{txs[9]})
	setParents(txs[11], []types.Txi{txs[10]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx, true)
	}

	// tx2 is bad, others are good
	verifier := GraphVerifier{
		Dag:    dag,
		TxPool: pool,
	}

	truth := []int{
		0, 1, 1, 1, 1,
		1, 1, 0, 1, 0,
		1, 1}

	for i, tx := range txs {
		judge(t, verifier.verifyA3(tx), truth[i] == 1, i)
	}

}

// A6: [My job] Node cannot reference two un-ordered nodes as its parents
func TestA6(t *testing.T) {
	pool := &dummyTxPoolParents{}
	pool.Init()
	dag := &dummyDag{}
	dag.init()

	addr1 := common.HexToAddress("0x0001")
	addr2 := common.HexToAddress("0x0002")
	addr3 := common.HexToAddress("0x0003")

	txs := []types.Txi{
		buildSeq(addr1, 1, 1),
		buildTx(addr1, 2),
		buildTx(addr1, 0),
		buildTx(addr2, 3),
		buildTx(addr3, 1),

		buildTx(addr2, 2),
		buildTx(addr1, 4),
		buildTx(addr3, 3),
		buildTx(addr2, 2),
	}

	setParents(txs[0], []types.Txi{})
	setParents(txs[1], []types.Txi{txs[0]})
	setParents(txs[2], []types.Txi{txs[1], txs[6]})
	setParents(txs[3], []types.Txi{txs[2], txs[7]})
	setParents(txs[4], []types.Txi{txs[3], txs[8]})

	setParents(txs[5], []types.Txi{txs[4]})
	setParents(txs[6], []types.Txi{txs[0]})
	setParents(txs[7], []types.Txi{txs[2]})
	setParents(txs[8], []types.Txi{txs[7]})

	for _, tx := range txs {
		pool.AddRemoteTx(tx, true)
	}

	// tx2 is bad, others are good
	verifier := GraphVerifier{
		Dag:    dag,
		TxPool: pool,
	}

	truth := []int{
		0, 1, 0, 1, 0,
		1, 1, 1, 1}

	for i, tx := range txs {
		judge(t, verifier.verifyA3(tx), truth[i] == 1, i)
	}

}

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
	v := TxFormatVerifier{NoVerifyMindHash: true, NoVerifyMaxTxHash: true}
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

func TestVerify(t *testing.T) {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	//_, priv := signer.RandomKeyPair()

	//pub, priv := signer.RandomKeyPair()
	//var txis types.Txis
	////var sigTerGets [][]byte
	//addr := pub.Address()
	//for i := 0; i < 10000; i++ {
	//	tx := tx_types.RandomTx()
	//	tx.From = &addr
	//	tx.Signature = signer.Sign(priv, tx.SignatureTargets()).Bytes
	//	tx.PublicKey = pub.Bytes
	//	txis = append(txis, tx)
	//}
	v := TxFormatVerifier{NoVerifyMindHash: true, NoVerifyMaxTxHash: true}
	//now := time.Now()
	//fmt.Println("start ", now)
	//for i, tx := range txis {
	//	ok := v.VerifySignature(tx)
	//	if !ok {
	//		t.Fatal(ok, tx, i)
	//	}
	//}
	//fmt.Println("used ", time.Since(now))
	//start := time.Now()
	newSigner := &TestSigner{}
	crypto.Signer = newSigner
	//fmt.Println(crypto.Signer.CanRecoverPubFromSig())
	//for i, tx := range txis {
	//	ok := v.VerifySignature(tx)
	//	if !ok {
	//		t.Fatal(ok, tx, i)
	//	}
	//}
	//fmt.Println("used ", time.Since(start))

	// test signature with recover id
	pub, priv := signer.RandomKeyPair()
	addr := signer.AddressFromPubKeyBytes(pub.Bytes)

	tx := tx_types.RandomTx()
	tx.From = &addr
	tx.To = common.HexToAddress("0xc1eb507017610d2134bc70146731d777a25c2889")
	tx.AccountNonce = 17320
	tx.Value, _ = math.NewBigIntFromString("894385949183117216", 10)
	tx.Data = common.Hex2Bytes("")
	tx.TokenId = 0

	sig := signer.Sign(priv, tx.SignatureTargets())
	tx.Signature = sig.Bytes
	//tx.Signature = common.Hex2Bytes("b08351552c718995e0d433850094e4e17eb2977bb479cae6488f6958c865c147313bc114142ae40ec90928466821a690b543b71489abea13cb6a8959e741fc211c")
	//tx.PublicKey = pub.Bytes

	fmt.Println("from: \t\t", tx.From.Hex())
	fmt.Println("to: \t\t", tx.To.Hex())
	fmt.Println("nonce: \t\t", tx.AccountNonce)
	fmt.Println("value: \t\t", tx.Value.String())
	fmt.Println(fmt.Sprintf("data: \t\t%x", tx.Data))
	fmt.Println("tokenID: \t", tx.TokenId)
	fmt.Println(fmt.Sprintf("sig: \t\t%x", []byte(tx.Signature)))
	fmt.Println(fmt.Sprintf("priv: \t\t%x", priv.Bytes))

	fmt.Println(fmt.Sprintf("sig targets: \t%x", tx.SignatureTargets()))

	if !v.VerifySignature(tx) {
		t.Fatalf("signature not correct")
	}
}

type TestSigner struct {
	crypto.SignerSecp256k1
}

func (s *TestSigner) CanRecoverPubFromSig() bool {
	return true
}
