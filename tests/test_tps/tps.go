package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/rpc"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
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
		from:=pub.Address()
		tx := tx_types.Tx{
			TxBase: types.TxBase{
				Type:         types.TxBaseTypeNormal,
				AccountNonce: uint64(i),
				PublicKey:    pub.Bytes[:],
			},
			From: &from ,
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

func newTransport() *http.Transport{
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
func newApp() app{
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

func testTPs() {
	var N= 100000
	var M= 99
	debug = false
	fmt.Println("started ", time.Now())

	var apps []app
	for i := 0; i < M; i++ {
		apps = append(apps, newApp())
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


func main() {
	debug = true
	a :=newApp()
	request := generateTokenPublishing()
	a.sendTx(&request,0,ipoUrl)
	return
}

type app struct {
	client      *http.Client
	requestChan chan *rpc.NewTxRequest
	quit        chan bool
}

var txurl = "http://172.28.152.101:11300/new_transaction"
var ipoUrl = "http://172.28.152.101:8000/token/NewPublicOffering"

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
			err := o.sendTx(data, i,txurl)
			if err != nil {
				logrus.WithError(err).Warnf("failed to send to ledger")
			}
		case <-o.quit:
			logrus.Info("OgProcessor stopped")
			return
		}
	}

}
func (a *app) sendTx(request interface{}, i int, url string ) error {
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


func generateTokenPublishing() rpc.NewPublicOfferingRequest{
	pub, priv := crypto.Signer.RandomKeyPair()
	from:= pub.Address()
	fmt.Println(pub.String(),priv.String(),from.String())
	value := math.NewBigInt(8888888)

		tx := tx_types.ActionTx{
			TxBase: types.TxBase{
				Type:         types.TxBaseAction,
				AccountNonce: uint64(1),
				PublicKey:    pub.Bytes[:],
			},
			Action:tx_types.ActionTxActionIPO,
			From: &from ,
			ActionData: &tx_types.PublicOffering{
				Value:value,
				EnableSPO:true,
				TokenName:"test_token",
			},
		}
		tx.Signature = crypto.Signer.Sign(priv, tx.SignatureTargets()).Bytes[:]
		v:=  og.TxFormatVerifier{}
		ok:= v.VerifySignature(&tx)
		if !ok {
			target := tx.SignatureTargets()
			fmt.Println(hexutil.Encode(target))
			panic("not ok")
		}
		request := rpc.NewPublicOfferingRequest{
			Nonce:     "1",
			From:      tx.From.Hex(),
			Value:     value.String(),
			Signature: tx.Signature.String(),
			Pubkey:    pub.String(),
			Action:    tx_types.ActionTxActionIPO,
			EnableSPO:true,
			TokenName:"test_token",
		}

	return request
}
