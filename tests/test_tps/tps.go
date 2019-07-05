package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/rpc"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func generateTxrequests(N int) []rpc.NewTxRequest {
	var requests []rpc.NewTxRequest
	oldpub, _ := crypto.Signer.RandomKeyPair()
	to := oldpub.Address().Hex()
	toAdd := oldpub.Address()
	pub, priv := crypto.Signer.RandomKeyPair()
	for i := 1; i < N; i++ {
		tx := types.Tx{
			TxBase: types.TxBase{
				Type:         types.TxBaseTypeNormal,
				AccountNonce: uint64(i),
				PublicKey:    pub.Bytes[:],
			},
			From:  pub.Address(),
			To:    toAdd,
			Value: math.NewBigInt(0),
		}
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
		//v:=  og.TxFormatVerifier{}
		//ok:= v.VerifySignature(&tx)
		//target := tx.SignatureTargets()
		//fmt.Println(hexutil.Encode(target))
		//if !ok {
		//	panic("not ok")
		//}
		request := rpc.NewTxRequest{
			Nonce:     "1",
			From:      tx.From.Hex(),
			To:        to,
			Value:     tx.Value.String(),
			Signature: tx.Signature.String(),
			Pubkey:    pub.String(),
		}
		requests = append(requests, request)
	}
	return requests
}

func main() {
	var N = 100000
	var M = 99
	debug = false
	fmt.Println("started ", time.Now())
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
	var apps []app
	for i := 0; i < M; i++ {
		a := app{
			client: &http.Client{
				Timeout:   time.Second * 10,
				Transport: transport,
			},
			requestChan: make(chan *rpc.NewTxRequest, 100),
			quit:        make(chan bool),
		}
		apps = append(apps, a)
	}
	requests := generateTxrequests(N)
	fmt.Println("gen txs ", time.Now(), len(requests))
	for i := 0; i < M; i++ {
		go apps[i].ConsumeQueue()
	}
	for i := range requests {
		j := i % M
		apps[j].requestChan <- &requests[i]
	}
	time.Sleep(time.Second * 100)
	for i := 0; i < M; i++ {
		close(apps[i].quit)
	}
	return
}

type app struct {
	client      *http.Client
	requestChan chan *rpc.NewTxRequest
	quit        chan bool
}

var url = "http://172.28.152.101:11300/new_transaction"

var debug bool

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
			err := o.sendTx(data, i)
			if err != nil {
				logrus.WithError(err).Warnf("failed to send to ledger")
			}
		case <-o.quit:
			logrus.Info("OgProcessor stopped")
			return
		}
	}

}
func (a *app) sendTx(request *rpc.NewTxRequest, i int) error {
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
	//fmt.Println(i, err, time.Since(now), str)
	return nil
}
