package main

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"time"
)

type  Monitor struct {
	Port  string `json:"port"`
	Peers []string `json:"peers"`
	SeqId  uint64 `json:"seq_id"`
	Err error
	id   int
}

type Statistics struct {
	PeersNum map[int] int
}

var fistPort = 7300

func main() {
	for {
		select {
		case <-time.After(8 * time.Second):
			go run()
		}
	}
}

type  Monitors struct {
	  Ms     []*Monitor `json:"monitors"`
}

func run () {
	var peerNum = 50
	var ms = make([]*Monitor,peerNum)
	mch := make(chan *Monitor ,peerNum)

	for i:=0;i<peerNum;i++ {
		go getRequest(i, mch)
	}
	var valid bool
	i:=0
	for {
		select {
		case data := <-mch:
			if data.Err==nil{
				valid = true
				d:=*data
				ms[data.id] = &d
			}else {
				//d:= Monitor{}
				//d.Port = fmt.Sprintf("%d", getPort(data.id))
				//ms[data.id]  = &d
			}
			i++
			if i==peerNum {
				goto Out
			}

		}
	}
	Out :
	if valid {
		fmt.Println("time now",time.Now().Format(time.RFC3339))
		var s Statistics
		s.PeersNum = make(map[int]int)
		for _,m:= range ms {
			if m==nil {
				continue
			}
			l := len( m.Peers)
			if v,ok:= s.PeersNum[l];ok {
				s.PeersNum[l] =  v+1
			}else {
				s.PeersNum[l] =1
			}
		}
		monitors:=Monitors{ms}
		data ,_:= json.MarshalIndent(&monitors,"","\t")
		sData ,_:= json.MarshalIndent(&s,"","\t")
		fmt.Println(string(data))
		fmt.Println(string(sData))
		fmt.Println("end \n\n")
	}
}

func getPort( id int ) int {
	return 	fistPort+id*10
}



func getRequest (id  int , ch chan *Monitor) {
	host := "http://192.168.45.143:"+ fmt.Sprintf("%d", getPort(id))
	req := httplib.NewBeegoRequest(host+"/monitor","GET")
	req.SetTimeout(5*time.Second,5*time.Second)
	var m Monitor
	err := req.ToJSON(&m)
	if err!=nil {
		m.Err =err
	}
	m.id = id
	ch <-&m
	return
}
