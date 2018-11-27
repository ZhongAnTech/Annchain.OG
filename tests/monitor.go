package main

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type  Monitor struct {
	Port  string `json:"port"`
	Peers []Peer `json:"peers"`
	SeqId  uint64 `json:"seq_id"`
	ShortId string `json:"short_id"`
	Err error
	id   int
}

type Peer struct {
	Addr    string `json:"addr"`
	ShortId string `json:"short_id"`
}

type Statistics struct {
	PeersNum map[int] int
}

var fistPort = 11300
var peerNum = 40
var ipsNum = 20

func main() {
	ips := GetIps()
	for {
		select {
		case <-time.After(10 * time.Second):
			go run(ips)
		}
	}
}

type  Monitors struct {
	  Ms     []*Monitor `json:"monitors,omitempty"`
}

func run (ips []string) {
	var ms = make([]*Monitor,peerNum*len(ips))
	mch := make(chan *Monitor ,peerNum*len(ips))
	for i, ip:= range ips {
		for j := 0; j < peerNum; j++ {
			go getRequest(ip,i*peerNum+j, j, mch)
		}
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
			if i==peerNum*len(ips) {
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

func GetIps () []string{
	dir,_ := os.Getwd()
	fName := fmt.Sprintf("%s/scripts/data/hosts",dir)
	f,err:= os.Open(fName)
	if err!=nil {
		panic(err)
	}
	defer f.Close()
	data,err := ioutil.ReadAll(f)
	if err!=nil {
		panic(err)
	}
	ips := strings.Split(string(data),"\n")
	return ips
}



func getRequest (ip string ,id ,portId int , ch chan *Monitor) {
	port := getPort(portId)
	host :=  fmt.Sprintf("http://%s:%d",ip,port )
	req := httplib.NewBeegoRequest(host+"/monitor","GET")
	req.SetTimeout(5*time.Second,5*time.Second)
	var m Monitor
	err := req.ToJSON(&m)
	if err!=nil {
		m.Err =err
	}
	m.id = id
	m.Port = fmt.Sprintf("%s:%s",ip, port)
	ch <-&m
	return
}
