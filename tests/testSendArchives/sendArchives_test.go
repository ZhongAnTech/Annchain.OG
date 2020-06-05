package testSendArchives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/annchain/OG/arefactor/og/types"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	var ar [][]byte
	for i := 0; i < 100000; i++ {
		ar = append(ar, types.RandomHash().ToBytes())
	}
	transport := &http.Transport{
		MaxIdleConnsPerHost: 15,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	a := app{
		client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: transport,
		},
	}
	for i, v := range ar {
		go a.sendArchiveData(v, i)
	}
	time.Sleep(time.Second * 100)
	return
}

type app struct {
	client *http.Client
}

var url = "http://172.28.152.101:8000/new_archive"

type txRequest struct {
	Data []byte `json:"data"`
}

func (a *app) sendArchiveData(data []byte, i int) {
	//req := httplib.NewBeegoRequest(url,"POST")
	//req.SetTimeout(time.Second*10,time.Second*10)
	tx := txRequest{
		Data: data,
	}
	data, err := json.Marshal(&tx)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(data)
	req, err := http.NewRequest("POST", url, r)

	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	//now := time.Now()
	defer resp.Body.Close()
	resDate, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	str := string(resDate)
	if err != nil {
		fmt.Println(i, str, err)
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode)
		return
	}
	//fmt.Println(i, err, time.Since(now), str)

}
