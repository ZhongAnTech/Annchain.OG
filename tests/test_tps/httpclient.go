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
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/rpc"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func newTransport() *http.Transport {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 15,
		Proxy:               http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return transport
}

func newApp() app {
	a := app{
		client: &http.Client{
			Timeout:   time.Second * 10,
			Transport: newTransport(),
		},
		requestChan: make(chan *rpc.NewTxRequest, 100),
		quit:        make(chan bool),
	}
	return a
}

type app struct {
	client      *http.Client
	requestChan chan *rpc.NewTxRequest
	quit        chan bool
}

func (o *app) ConsumeQueue() {
	i := 0
	for {
		logrus.WithField("size", len(o.requestChan)).Debug("og queue size")
		select {
		case data := <-o.requestChan:
			i++
			if debug {
				fmt.Println(data)
			}
			err := o.sendTx(data, i, txurl)
			if err != nil {
				logrus.WithError(err).Warnf("failed to send to ledger")
			}
		case <-o.quit:
			logrus.Info("OgProcessor stopped")
			return
		}
	}

}

func (a *app) sendTx(request interface{}, i int, url string) error {
	//req := httplib.NewBeegoRequest(url,"POST")
	//req.SetTimeout(time.Second*10,time.Second*10)
	data, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}
	r := bytes.NewReader(data)
	req, err := http.NewRequest("POST", url, r)

	resp, err := a.client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
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
		return err
	}
	if resp.StatusCode != 200 {
		//panic( resp.StatusCode)
		fmt.Println(resp.StatusCode)
		return errors.New(resp.Status)
	}
	if debug {
		fmt.Println(i, err, str)
	}
	return nil
}
