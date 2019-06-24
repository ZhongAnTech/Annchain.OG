package testSendArchives

import (
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"github.com/annchain/OG/types"
	"testing"
	"time"
)

func TestSend(t *testing.T){
	var ar [][]byte
   for i:=0;i<10;i++ {
      ar = append(ar,types.RandomHash().ToBytes())
   }
   for i,v := range ar {
   	 go sendArchiveData(v,i)
   }
   time.Sleep(time.Second*100)
	return
}
var ulr = "http://172.28.152.101:8000/new_archive"

type txRequest struct {
	Data []byte `json:"data"`
}

func sendArchiveData (data[]byte,i int ) {
	req := httplib.NewBeegoRequest(ulr,"POST")
	req.SetTimeout(time.Second*10,time.Second*10)
	tx:= txRequest{
		Data:data,
	}
	req,err := req.JSONBody(&tx)
	if err!=nil {
		panic(err)
	}
	now := time.Now()
	str,err:=req.String()
	if err!=nil {
		fmt.Println(i,str,err)
	}
	fmt.Println(i, err, time.Since(now))

}
