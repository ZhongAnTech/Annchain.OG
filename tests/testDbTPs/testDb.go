package main

import (
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/core"
	"github.com/annchain/OG/core/state"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func generateTxs (totalHeight int,txnum int  ) []*core.ConfirmBatch{
	var height uint64
	var batchs  []*core.ConfirmBatch
	for j:=0;j<totalHeight;j++ {
		var txis types.Txis
		pub, priv := crypto.Signer.RandomKeyPair()
		for i := txnum; i > 0; i-- {
			tx := types.RandomTx()
			tx.Value= math.NewBigInt(0)
			tx.PublicKey = pub.Bytes[:]
			tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
			txis = append(txis, tx)
		}
		seq := types.RandomSequencer()
		seq.PublicKey = pub.Bytes[:]
		seq.Signature = crypto.Signer.Sign(priv, seq.SignatureTargets()).Bytes[:]
		batch := &core.ConfirmBatch{
			Seq: seq,
			Txs: txis,
		}
		height++
		batch.Seq.Height = height
		batchs = append(batchs, batch)
	}
	return batchs
}




func main(){
	go func() {
		http.ListenAndServe("0.0.0.0:"+"9095", nil)
	}()
	db,err := ogdb.NewLevelDB("datadir",512,512)
	if err !=nil{
		panic(err)
	}
    dag,err := core.NewDag(core.DagConfig{GenesisPath:"genesis.json"},state.DefaultStateDBConfig(),db,nil	)
    if err!=nil {
    	panic(err)
	}
    fmt.Println("dag init done",time.Now())
    height := 30
    txnum :=10000
    batchs := generateTxs(height,txnum)
	fmt.Println("gen tx done",time.Now())
    logrus.SetLevel(logrus.WarnLevel)
    start :=time.Now()
    for i:= range  batchs {
		local :=time.Now()
    	err = dag.Push(batchs[i])
    	if err!=nil {
    		panic(err)
		}
    	since:= time.Since(local)
    	tps := int64(txnum)* int64( time.Second)/since.Nanoseconds()
    	fmt.Println("used time for push ",tps ,batchs[i].Seq,since.String())
	}
	since:= time.Since(start)
	tps := int64( txnum)* int64( time.Second)/since.Nanoseconds()
	fmt.Println("used time for all ", time.Since(start),tps )

}
