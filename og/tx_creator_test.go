package og

import (
	"testing"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/og/miner"
	"github.com/stretchr/testify/assert"
	"github.com/sirupsen/logrus"
	"time"
	"math/rand"
	"github.com/annchain/OG/common/math"
)

type dummyTxPoolRandomTx struct {
}

func (p *dummyTxPoolRandomTx) GetRandomTips(n int) (v []types.Txi) {
	for i := 0; i < n; i++ {
		v = append(v, types.RandomTx())
	}
	return
}

type dummyTxPoolMiniTx struct {
	poolMap map[types.Hash]types.Txi
	tipsMap map[types.Hash]types.Txi
}

func (d *dummyTxPoolMiniTx) Init(){
	d.poolMap = make(map[types.Hash]types.Txi)
	d.tipsMap = make(map[types.Hash]types.Txi)
}

// generate [count] unique random number within range [0, upper)
// if count > upper, use all available indices
func generateRandomIndices(count int, upper int) []int {
	if count > upper {
		count = upper
	}
	// avoid dup
	generated := make(map[int]struct{})
	for count > len(generated) {
		i := rand.Intn(upper)
		if _, ok := generated[i]; ok {
			continue
		}
		generated[i] = struct{}{}
	}
	arr := make([]int, 0, len(generated))
	for k := range generated {
		arr = append(arr, k)
	}
	return arr
}

func (p *dummyTxPoolMiniTx) GetRandomTips(n int) (v []types.Txi) {
	indices := generateRandomIndices(n, len(p.tipsMap))
	// slice of keys
	var keys []types.Hash
	for k := range p.tipsMap {
		keys = append(keys, k)
	}
	for i := range indices {
		v = append(v, p.tipsMap[keys[i]])
	}
	return v
}

func (p *dummyTxPoolMiniTx) Add(v types.Txi) {
	p.tipsMap[v.GetTxHash()] = v

	for _, parentHash := range v.Parents() {
		logrus.Infof("Parent: %s", parentHash.Hex())
		if vp, ok := p.tipsMap[parentHash]; ok {
			delete(p.tipsMap, parentHash)
			p.poolMap[parentHash] = vp
		}
	}
	logrus.Infof("Added tx %s to tip. Current pool size: tips: %d pool: %d",
		v.GetTxHash().Hex(), len(p.tipsMap), len(p.poolMap))
}

func Init() *TxCreator {
	txc := TxCreator{
		Signer:             &crypto.SignerEd25519{},
		TipGenerator:       &dummyTxPoolRandomTx{},
		Miner:              &miner.PoWMiner{},
		MaxConnectingTries: 100,
		MaxTxHash:          types.HexToHash("0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaxMinedHash:       types.HexToHash("0x00000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}
	return &txc
}

func TestTxCreator(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	txc := Init()
	tx := txc.TipGenerator.GetRandomTips(1)[0].(*types.Tx)
	_, priv, err := txc.Signer.RandomKeyPair()
	assert.NoError(t, err)
	time1 := time.Now()
	txSigned := txc.NewSignedTx(tx.From, tx.To, tx.Value, tx.AccountNonce, priv)
	logrus.Infof("Total time for Signing: %d ns", time.Since(time1).Nanoseconds())
	ok := txc.SealTx(txSigned)
	logrus.Infof("Result: %t %v", ok, txSigned)
}

func TestSequencerCreator(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	txc := Init()
	_, priv, err := txc.Signer.RandomKeyPair()
	assert.NoError(t, err)
	time1 := time.Now()

	// for copy
	randomSeq := types.RandomSequencer()

	txSigned := txc.NewSignedSequencer(randomSeq.Id, randomSeq.ContractHashOrder, randomSeq.AccountNonce, priv)
	logrus.Infof("Total time for Signing: %d ns", time.Since(time1).Nanoseconds())
	ok := txc.SealTx(txSigned)
	logrus.Infof("Result: %t %v", ok, txSigned)
}

func sampleTxi(selfHash string, parentsHash []string, baseType types.TxBaseType) types.Txi {

	tx := &types.Tx{TxBase: types.TxBase{
		ParentsHash: []types.Hash{},
		Type:        types.TxBaseTypeNormal,
		Hash:        types.HexToHash(selfHash),
	},
	}
	for _, h := range parentsHash {
		tx.ParentsHash = append(tx.ParentsHash, types.HexToHash(h))
	}
	return tx
}


func TestBuildDag(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	pool := &dummyTxPoolMiniTx{}
	pool.Init()
	txc := TxCreator{
		Signer:             &crypto.SignerEd25519{},
		TipGenerator:       pool,
		Miner:              &miner.PoWMiner{},
		MaxConnectingTries: 10,
		MaxTxHash:          types.HexToHash("0x0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		MaxMinedHash:       types.HexToHash("0x000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
	}

	_, privateKey, _ := txc.Signer.RandomKeyPair()

	txs := []types.Txi{
		txc.NewSignedSequencer(0, []types.Hash{}, 0, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x01"), types.HexToAddress("0x02"), math.NewBigInt(10), 0, privateKey),
		txc.NewSignedSequencer(1, []types.Hash{}, 1, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x02"), types.HexToAddress("0x03"), math.NewBigInt(9), 0, privateKey),
		txc.NewSignedTx(types.HexToAddress("0x03"), types.HexToAddress("0x04"), math.NewBigInt(8), 0, privateKey),
		txc.NewSignedSequencer(2, []types.Hash{}, 2, privateKey),
	}

	txs[0].GetBase().Hash = txs[0].CalcTxHash()
	pool.Add(txs[0])
	for i := 1 ; i< len(txs) ; i++{
		if ok := txc.SealTx(txs[i]); ok{
			pool.Add(txs[i])
		}
	}
}
