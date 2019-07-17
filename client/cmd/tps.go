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
	"encoding/binary"
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
	"strings"
	"sync"
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
	num  uint16
	times uint16
	accountNum uint16
	ipports string
)

func tpsInit() {
	tpsCmd.AddCommand(tpsGenCmd, tpsSendTxCmd)
	tpsCmd.PersistentFlags().Uint16VarP(&num, "num", "n", 1000,"num 1000")
	tpsCmd.PersistentFlags().Uint16VarP(&accountNum, "accounts_num", "a", 4,"accounts_num 1000")
	tpsCmd.PersistentFlags().StringVarP(&ipports, "ips", "i", "","accounts_num 1000")
	tpsGenCmd.PersistentFlags().Uint16VarP(&times, "times", "t", 1000,"times 1000")
}

func tepsDataGen( threadNum uint16 , db ogdb.Database ,total uint16) {
	_, priv := crypto.Signer.RandomKeyPair()
	requester :=tx_client.NewRequestGenerator(priv)
	to:= common.RandomAddress()
	requester.Nodebug = true
	fmt.Println("will generate tx ",num ," * ", times )
	for i:= uint16(0);i<total;i++ {
		var reqs rpc.NewTxsRequests
		for j:= uint16(0);j<num;j++ {
			txReq:= requester.NormalTx(0,uint64(i*num+1+j),to ,math.NewBigInt(0))
			reqs.Txs = append(reqs.Txs,txReq )
		}
		data,err := reqs.MarshalMsg(nil)
		panicIfError(err, "marshal err")
		key := makeKey(threadNum,i)
		err = db.Put(key,data)
		panicIfError(err, "db err")
		fmt.Println("gen tx ",i, threadNum)
	}
}

func makeKey(i ,j uint16) []byte 	{
	data1 := make([]byte,2)
	binary.BigEndian.PutUint16(data1,i)
	data2 := make([]byte,2)
	binary.BigEndian.PutUint16(data2,j)
	return  append(data1,data2...)
}


func tpsGen(cmd *cobra.Command, args []string) {
	db ,err := generateDb()
	panicIfError(err ,"")
	defer db.Close()
	start := time.Now()
	//mp:= runtime.GOMAXPROCS(0)
	mp:= int(accountNum)
	var wg = &sync.WaitGroup{}
	wg.Wait()
	for i:=0;i<mp;i++ {
		wg.Add(1)
		go func(k uint16 ) {
			tepsDataGen(k,db,times/ uint16(mp))
			wg.Done()
		}(uint16(i))
	}
	wg.Wait()
	fmt.Println("used time for generating txs ", time.Since(start), num*times)

}

func tpsSend(cmd *cobra.Command, args []string) {
	db ,err := generateDb()
	panicIfError(err ,"")
	defer db.Close()
	start := time.Now()
	//mp:= runtime.GOMAXPROCS(0)
	//mp:= int(accountNum)
	ipportList := strings.Split(ipports,";")
	if len(ipportList) == 0 {
		fmt.Println("need ips and ports , for example : 127.0.0.1:8000;127.0.0.1:8010")
		return
	}
	if int( accountNum) < len(ipportList) {
		fmt.Println("need more accounts")
		return
	}
	var wg = &sync.WaitGroup{}
	hostNum :=  uint16(len(ipportList))
	perHost :=  accountNum/ hostNum
	for i:=uint16(0) ;i<hostNum;i++ {
		for j:=uint16(0) ;j<perHost ;j++ {
			wg.Add(1)
			go func(k uint16, host string ) {
				tpsSendData(k, db, host)
				wg.Done()
			}(i*perHost+j,ipportList[i])
		}
	}
	wg.Wait()
	fmt.Println("used time for generating txs ", time.Since(start), num*times)

}

func tpsSendData(threadNum uint16,db ogdb.Database, host string  ) {
	txClient  := tx_client.NewTxClientWIthTimeOut(host,false, time.Second*120)
	Max:=  1000000
	for i:= 0; i<Max;i++ {
		var reqs rpc.NewTxsRequests
		key := makeKey(threadNum,uint16(i))
		data,err := db.Get(key)
		if err != nil || len(data) ==0 {
			fmt.Println("read data err ",err,i )
			break
		}
		_, err = reqs.UnmarshalMsg(data)
		panicIfError(err, "unmarshal err")
		fmt.Println("sending  data " ,i ,threadNum, len(reqs.Txs))
		resp,err := txClient.SendNormalTxs(&reqs)
		//panicIfError(err, resp)
		if err!=nil {
			fmt.Println(err,resp)
			return 
		}
	}
}

func generateDb() (ogdb.Database, error) {
	path := io.FixPrefixPath(viper.GetString("./"), "test_tps_db")
	return ogdb.NewLevelDB(path, 512, 512)

}