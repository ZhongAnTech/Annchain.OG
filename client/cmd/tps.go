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

package cmd

import (
	"fmt"
	"github.com/annchain/OG/client/tx_client"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/io"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/ogdb"
	"github.com/annchain/OG/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"time"
)


var (
	tpsCmd = &cobra.Command{
		Use:   "tps",
		Short: "tps",
	}
	tpsGenCmd = &cobra.Command{
		Use:   "gen",
		Short: "generate tx",
		Run:   tpsGen,
	}

	tpsSendTxCmd = &cobra.Command{
		Use:   "send",
		Short: "send",
		Run:   tpsSend,
	}
	num  uint64
	times uint64
)

func tpsInit() {
	tpsCmd.AddCommand(tpsGenCmd, tpsSendTxCmd)
	tpsCmd.PersistentFlags().Uint64VarP(&num, "num", "n", 1000,"num 1000")
	tpsGenCmd.PersistentFlags().Uint64VarP(&times, "times", "t", 1000,"times 1000")
}


func tpsGen(cmd *cobra.Command, args []string) {
	_, priv := crypto.Signer.RandomKeyPair()
	requester :=tx_client.NewRequestGenerator(priv)
	requester.Nodebug = true
	to:= common.RandomAddress()
	db ,err := generateDb()
	panicIfError(err ,"")
	start := time.Now()
	defer db.Close()
	fmt.Println("will generate tx ",num ," * ", times )
	for i:= uint64(0);i<times;i++ {
		var reqs rpc.NewTxsRequests
		for j:= uint64(0);j<num;j++ {
			txReq:= requester.NormalTx(0,i*num+1+j,to ,math.NewBigInt(0))
			reqs.Txs = append(reqs.Txs,txReq )
		}
		data,err := reqs.MarshalMsg(nil)
		panicIfError(err, "marshal err")
		key := common.ByteInt32(int32(i))
		err = db.Put(key,data)
		panicIfError(err, "db err")
		fmt.Println("gen tx ",i)
	}
	fmt.Println("used time for generating txs ", time.Since(start), num*times)
}

func tpsSend(cmd *cobra.Command, args []string) {
	txClient  := tx_client.NewTxClientWIthTimeOut(Host,true, time.Second*20)
	db ,err := generateDb()
	panicIfError(err ,"")
	defer db.Close()
	Max:=  1000000
	for i:= 0; i<Max;i++ {
		var reqs rpc.NewTxsRequests
		data,err := db.Get(common.ByteInt32(int32(i)))
		if err!=nil || len(data) ==0 {
			fmt.Println("read data err ",err,i )
			break
		}
		_, err = reqs.UnmarshalMsg(data)
		panicIfError(err, "unmarshal err")
		fmt.Println("sending  data " ,i , len(reqs.Txs))
		resp,err := txClient.SendNormalTxs(&reqs)
		panicIfError(err, "send tx err")
		_ = resp
	}
}


func generateDb() (ogdb.Database, error) {
	path := io.FixPrefixPath(viper.GetString("./"), "test_tps_db")
	return ogdb.NewLevelDB(path, 512, 512)

}