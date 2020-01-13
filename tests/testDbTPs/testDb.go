package main

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og/types"
	archive2 "github.com/annchain/OG/og/types/archive"
	core2 "github.com/annchain/OG/ogcore/ledger"
	"github.com/annchain/OG/ogcore/state"

	"github.com/annchain/OG/ogdb"
	"github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"time"
)

var archive bool

func generateTxs(height uint64, totalHeight int, txnum int) []*core2.ConfirmBatch {
	var batchs []*core2.ConfirmBatch
	for j := 0; j < totalHeight; j++ {
		pub, priv := crypto.Signer.RandomKeyPair()
		var txis types.Txis
		for i := 0; i < txnum; i++ {
			if archive {
				ar := archive.RandomArchive()
				ar.Data = append(ar.Data, pub.KeyBytes[:]...)
				ar.Data = append(ar.Data, pub.KeyBytes[:]...)
				ar.Data = append(ar.Data, pub.KeyBytes[:]...)
				txis = append(txis, ar)
			} else {
				tx := archive2.RandomTx()
				tx.Value = math.NewBigInt(0)
				tx.PublicKey = pub.KeyBytes[:]
				tx.From = pub.Address()
				tx.AccountNonce = uint64(i) + 1
				tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).SignatureBytes[:]
				txis = append(txis, tx)
			}
		}
		seq := types.RandomSequencer()
		seq.PublicKey = pub.KeyBytes[:]
		seq.Signature = crypto.Signer.Sign(priv, seq.SignatureTargets()).SignatureBytes[:]
		batch := &core2.ConfirmBatch{
			Seq: seq,
			Txs: txis,
		}
		height++
		batch.Seq.Height = height
		batchs = append(batchs, batch)
	}
	return batchs
}

func main() {
	archive = false
	go func() {
		http.ListenAndServe("0.0.0.0:"+"9095", nil)
	}()
	db, err := ogdb.NewLevelDB("datadir", 512, 512)
	if err != nil {
		panic(err)
	}
	dag, err := core2.NewDag(core2.DagConfig{GenesisGenerator: &core2.ConfigFileGenesisGenerator{Path: "genesis.json"}},
		state.DefaultStateDBConfig(), db, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("dag init done", time.Now())
	totalHeight := 35
	txnum := 10000
	batchs := generateTxs(dag.LatestSequencer().Height, totalHeight, txnum)
	fmt.Println("gen tx done", time.Now())
	logrus.SetLevel(logrus.WarnLevel)
	start := time.Now()
	for i := range batchs {
		local := time.Now()
		batch := batchs[i]
		err = dag.Push(batch)
		if err != nil {
			panic(err)
		}
		since := time.Since(local)
		tps := int64(txnum) * int64(time.Second) / since.Nanoseconds()
		fmt.Println("used time for push ", tps, batch.Seq, since.String())
	}
	dag.Stop()
	since := time.Since(start)
	tps := int64(totalHeight*txnum) * int64(time.Second) / since.Nanoseconds()
	fmt.Println("used time for all ", time.Since(start), tps)

}
