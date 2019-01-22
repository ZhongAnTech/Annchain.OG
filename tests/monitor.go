package main

import (
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/client/httplib"
	"github.com/annchain/OG/rpc"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type Monitor struct {
	Port    string         `json:"port"`
	Peers   []rpc.Peer     `json:"peers"`
	SeqId   uint64         `json:"seq_id"`
	ShortId string         `json:"short_id"`
	Status  rpc.SyncStatus `json:"status"`
	Err     error
	Id      int
	Tps     *rpc.Tps `json:"tps"`
}

type Statistics struct {
	PeersNum map[int]int
}

var fistPort = 11300
var peerNum = 3
var ipsNum = 1

func main() {
	ips := GetIps()
	for {
		select {
		case <-time.After(2 * time.Second):
			go run(ips)
		}
	}
}

type Monitors struct {
	Ms []*Monitor `json:"monitors,omitempty"`
}

func run(ips []string) {
	var ms = make([]*Monitor, peerNum*len(ips))
	mch := make(chan *Monitor, peerNum*len(ips))
	for i, ip := range ips {
		for j := 0; j < peerNum; j++ {
			go getRequest(ip, i*peerNum+j, j, mch)
		}
	}
	var valid bool
	i := 0
	for {
		select {
		case data := <-mch:
			if data.Err == nil {
				valid = true
				d := *data
				ms[data.Id] = &d
			} else {
				//d:= Monitor{}
				//d.Port = fmt.Sprintf("%d", getPort(data.id))
				//ms[data.id]  = &d
			}
			i++
			if i == peerNum*len(ips) {
				goto Out
			}

		}
	}
Out:
	if valid {
		fmt.Println("time now", time.Now().Format(time.RFC3339))
		var s Statistics
		s.PeersNum = make(map[int]int)
		for _, m := range ms {
			if m == nil {
				continue
			}
			l := len(m.Peers)
			if v, ok := s.PeersNum[l]; ok {
				s.PeersNum[l] = v + 1
			} else {
				s.PeersNum[l] = 1
			}
			//m.Peers = nil
		}
		monitors := Monitors{ms}
		data, _ := json.MarshalIndent(&monitors, "", "\t")
		sData, _ := json.MarshalIndent(&s, "", "\t")
		fmt.Println(string(data))
		fmt.Println(string(sData))
		fmt.Println("end \n\n")
	}
}

func getPort(id int) int {
	return fistPort + id*10
}

func GetIps() []string {
	return  []string{"192.168.45.145"}
	dir, _ := os.Getwd()
	fName := fmt.Sprintf("%s/scripts/data/hosts", dir)
	f, err := os.Open(fName)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	ips := strings.Split(string(data), "\n")
	if len(ips) > ipsNum {
		ips = ips[:ipsNum]
	}
	return ips
}

func getRequest(ip string, id, portId int, ch chan *Monitor) {
	port := getPort(portId)
	host := fmt.Sprintf("http://%s:%d", ip, port)
	req := httplib.NewBeegoRequest(host+"/monitor", "GET")
	req.SetTimeout(8*time.Second, 8*time.Second)
	var m Monitor
	err := req.ToJSON(&m)
	if err != nil {
		m.Err = err
	}
	m.Id = id
	m.Port = fmt.Sprintf("%s:%d", ip, port)
	ch <- &m
	return
}
