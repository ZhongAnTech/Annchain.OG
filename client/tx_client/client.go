// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by TxClientlicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tx_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func newTransport(timeOut time.Duration) *http.Transport {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 15,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeOut,
			KeepAlive: timeOut * 3,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       timeOut * 9,
		TLSHandshakeTimeout:   timeOut,
		ExpectContinueTimeout: timeOut / 10,
	}
	return transport
}

func NewTxClient(Host string, debug bool) TxClient {
	a := TxClient{
		httpClient: &http.Client{
			Timeout:   time.Second * 10,
			Transport: newTransport(time.Second * 10),
		},
		requestChan: make(chan *rpc.NewTxRequest, 100),
		quit:        make(chan bool),
		Host:        Host,
		Debug:       debug,
	}
	return a
}

func NewTxClientWIthTimeOut(Host string, debug bool, timeOut time.Duration) TxClient {
	a := TxClient{
		httpClient: &http.Client{
			Timeout:   timeOut,
			Transport: newTransport(timeOut),
		},
		requestChan: make(chan *rpc.NewTxRequest, 100),
		quit:        make(chan bool),
		Host:        Host,
		Debug:       debug,
	}
	return a
}

type TxClient struct {
	httpClient  *http.Client
	requestChan chan *rpc.NewTxRequest
	quit        chan bool
	Host        string
	Debug       bool
}

func (a *TxClient) StartAsyncLoop() {
	go a.ConsumeQueue()
}

func (a *TxClient) Stop() {
	close(a.quit)
}

func (A *TxClient) SendAsyncTx(Req *rpc.NewTxRequest) {
	A.requestChan <- Req
}

func (o *TxClient) ConsumeQueue() {
	i := 0
	for {
		logrus.WithField("size", len(o.requestChan)).Debug("og queue size")
		select {
		case data := <-o.requestChan:
			i++
			if o.Debug {
				fmt.Println(data)
			}
			resp, err := o.SendNormalTx(data)
			if err != nil {
				logrus.WithField("resp", resp).WithError(err).Warnf("failed to send to ledger")
			}
		case <-o.quit:
			logrus.Info("OgProcessor stopped")
			return
		}
	}

}

func (a *TxClient) SendNormalTx(request *rpc.NewTxRequest) (string, error) {
	return a.sendTx(request, "new_transaction", "POST")
}

func (a *TxClient) SendNormalTxs(request *rpc.NewTxsRequests) (string, error) {
	return a.sendTxs(request, "new_transactions", "POST")
}

func (a *TxClient) SendTokenIPO(request *rpc.NewPublicOfferingRequest) (string, error) {
	return a.sendTx(request, "token", "POST")
}

func (a *TxClient) SendTokenSPO(request *rpc.NewPublicOfferingRequest) (string, error) {
	return a.sendTx(request, "token", "PUT")
}

func (a *TxClient) SendTokenDestroy(request *rpc.NewPublicOfferingRequest) (string, error) {
	return a.sendTx(request, "token", "DELETE")
}

func (a *TxClient) sendTx(request interface{}, uri string, methd string) (string, error) {
	//req := httplib.NewBeegoRequest(url,"POST")
	//req.SetTimeout(time.Second*10,time.Second*10)
	data, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(data)
	url := a.Host + "/" + uri
	req, err := http.NewRequest(methd, url, r)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		//fmt.Println(err)
		return "", err
	}
	//now := time.Now()
	defer resp.Body.Close()
	resDate, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	str := string(resDate)
	if err != nil {
		fmt.Println(str, err)
		return "", err
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode)
		return "", errors.New(resp.Status)
	}
	var respStruct struct {
		Data string `json:"data"`
	}
	err = json.Unmarshal(resDate, &respStruct)
	if err != nil {
		//fmt.Println(str, err)
		return "", err
	}
	if a.Debug {
		fmt.Println(respStruct.Data)
	}
	return respStruct.Data, nil
}

func (a *TxClient) sendTxs(request interface{}, uri string, methd string) (string, error) {
	//req := httplib.NewBeegoRequest(url,"POST")
	//req.SetTimeout(time.Second*10,time.Second*10)
	data, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(data)
	url := a.Host + "/" + uri
	req, err := http.NewRequest(methd, url, r)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		//fmt.Println(err)
		return "", err
	}
	//now := time.Now()
	defer resp.Body.Close()
	resDate, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	str := string(resDate)
	if err != nil {
		fmt.Println(str, err)
		return "", err
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode, str)
		return "", errors.New(resp.Status)
	}
	//var respStruct struct{
	//	Data common.Hashes  `json:"data"`
	//}
	if a.Debug {
		fmt.Println(str)
	}
	return str, nil
}

func (a *TxClient) GetNonce(addr common.Address) (nonce uint64, err error) {
	uri := fmt.Sprintf("query_nonce?address=%s", addr.Hex())
	url := a.Host + "/" + uri
	req, err := http.NewRequest("GET", url, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		//fmt.Println(err)
		return 0, err
	}
	//now := time.Now()
	defer resp.Body.Close()
	resDate, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	str := string(resDate)
	if err != nil {
		fmt.Println(str, err)
		return 0, err
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode)
		return 0, err
	}
	var nonceResp struct {
		Data uint64 `json:"data"`
	}
	err = json.Unmarshal(resDate, &nonceResp)
	if err != nil {
		//fmt.Println("encode nonce errror ", err)
		return 0, err
	}
	return nonceResp.Data, nil
}

//type TokenList map[string]string

func (a *TxClient) GetTokenList() (TokenList string, err error) {
	url := a.Host + "/" + "token/list"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		//fmt.Println(err)
		return "", err
	}
	//now := time.Now()
	defer resp.Body.Close()
	resDate, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	str := string(resDate)
	if err != nil {
		fmt.Println(str, err)
		return "", err
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode)
		return "", errors.New(resp.Status)
	}
	return str, nil
}
