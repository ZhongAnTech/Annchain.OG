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
package annsensus

import (
	"bytes"
	"fmt"
	"github.com/annchain/OG/account"
	"testing"
	"time"

	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/filename"
	"github.com/annchain/OG/common/gcache"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/sirupsen/logrus"
)

type TestAnnSensus struct {
	Id int
	*AnnSensus
}

type p2pMsg struct {
	data    []byte
	msgType og.MessageType
}

type TestHub struct {
	Id              int
	Peers           []int
	sendMsgToChan   sendMsgToChanFunc
	sendMsgByPubKey sendMsgByPubKeyFunc
	OutMsg          chan p2pMsg
	quit            chan struct{}
	As              *AnnSensus
	msgCache        gcache.Cache
}

type sendMsgToChanFunc func(id int, mdg TestMsg)
type sendMsgByPubKeyFunc func(pub *crypto.PublicKey, msg TestMsg)

func newtestHub(id int, peers []int, sendMsgToChan sendMsgToChanFunc, sendMsgByPubKey sendMsgByPubKeyFunc, as *AnnSensus) *TestHub {
	return &TestHub{
		Id:              id,
		Peers:           peers,
		OutMsg:          make(chan p2pMsg, 100),
		sendMsgToChan:   sendMsgToChan,
		sendMsgByPubKey: sendMsgByPubKey,
		quit:            make(chan struct{}),
		As:              as,
		msgCache:        gcache.New(1000).Expiration(time.Minute).Simple().Build(),
	}
}

func (t *TestHub) BroadcastMessage(messageType og.MessageType, message types.Message) {
	var sent bool
	for _, peer := range t.Peers {
		tMsg := TestMsg{
			MessageType: messageType,
			Message:     message,
			From:        t.Id,
		}
		t.sendMsgToChan(peer, tMsg)
		if !sent {
			hash := tMsg.GetHash()
			t.msgCache.Set(hash, struct{}{})
			sent = true
		}
		//logrus.WithField("me ", t.Id).WithField("to peer ", peer).WithField("type ", messageType).Trace("send msg")
	}
}

func (t *TestHub) SendToAnynomous(messageType og.MessageType, message types.Message, anyNomousPubKey *crypto.PublicKey) {
	tMsg := TestMsg{
		MessageType: messageType,
		Message:     message,
		From:        t.Id,
	}
	t.sendMsgByPubKey(anyNomousPubKey, tMsg)
	hash := tMsg.GetHash()
	t.msgCache.Set(hash, struct{}{})
}

func (t *TestHub) loop() {
	elog := logrus.WithField("me", t.Id)
	for {
		select {
		case pMsg := <-t.OutMsg:
			var msg TestMsg
			switch pMsg.msgType {
			case og.MessageTypeConsensusDkgDeal:
				msg.MessageType = pMsg.msgType
				msg.Message = &types.MessageConsensusDkgDeal{}
			case og.MessageTypeConsensusDkgDealResponse:
				msg.MessageType = pMsg.msgType
				msg.Message = &types.MessageConsensusDkgDealResponse{}
			case og.MessageTypeConsensusDkgSigSets:
				msg.MessageType = pMsg.msgType
				msg.Message = &types.MessageConsensusDkgSigSets{}
			default:
				elog.Warn(pMsg, " unkown meg type will panic ")
				panic(pMsg)
			}
			_, err := msg.UnmarshalMsg(pMsg.data)
			if err != nil {
				panic(err)
			}
			time.Sleep(10 * time.Millisecond)
			hash := msg.GetHash()
			switch msg.MessageType {
			case og.MessageTypeConsensusDkgDeal:
				request := msg.Message.(*types.MessageConsensusDkgDeal)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.Data)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate dkg msg")
					continue
				}
				go t.As.HandleConsensusDkgDeal(request, fmt.Sprintf("%d", msg.From))
			case og.MessageTypeConsensusDkgDealResponse:
				request := msg.Message.(*types.MessageConsensusDkgDealResponse)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.Data)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate response  msg")
					continue
				}
				go t.As.HandleConsensusDkgDealResponse(request, fmt.Sprintf("%d", msg.From))
			case og.MessageTypeConsensusDkgSigSets:
				request := msg.Message.(*types.MessageConsensusDkgSigSets)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.PkBls)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate response  msg")
					continue
				}
				go t.As.HandleConsensusDkgSigSets(request, fmt.Sprintf("%d", msg.From))
			default:
				elog.Info("never come here , msg loop ")
				return
			}
			elog.WithField("msg ", msg).Debug("i got a msg")
			t.msgCache.Set(hash, struct{}{})
		case <-t.quit:
			elog.Debug("stopped")
			return
		}
	}
}

func TestDKGMain(t *testing.T) {
	logInit()
	var Anns []TestAnnSensus
	sendMsgToChan := func(id int, msg TestMsg) {
		data, err := msg.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		pMsg := p2pMsg{data: data, msgType: msg.MessageType}
		Anns[id].Hub.(*TestHub).OutMsg <- pMsg
		var resp *types.MessageConsensusDkgDealResponse
		if msg.MessageType == og.MessageTypeConsensusDkgDealResponse {
			resp = msg.Message.(*types.MessageConsensusDkgDealResponse)
		}
		logrus.WithField("me ", msg.From).WithField("to peer ", id).WithField("type ",
			msg.MessageType).WithField("msg ", resp).WithField("len ", len(data)).Trace("send msg")
		return
	}
	sendMsgByPubKey := func(pub *crypto.PublicKey, msg TestMsg) {
		data, err := msg.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		pMsg := p2pMsg{data: data, msgType: msg.MessageType}
		for j := 0; j < 4; j++ {
			if bytes.Equal(Anns[j].MyAccount.PublicKey.Bytes, pub.Bytes) {
				Anns[j].Hub.(*TestHub).OutMsg <- pMsg
				logrus.WithField("from peer", msg.From).WithField("to peer ", j).WithField("type ",
					msg.MessageType).Trace("send msg enc")
				return
			}
		}
		logrus.Warn("not found for pubkey ", pub)
		return
	}

	for j := 0; j < 4; j++ {
		as := NewAnnSensus(crypto.CryptoTypeSecp256k1, true, 4, 4, time.Second, nil, nil, nil)
		as.id = j
		a := TestAnnSensus{
			Id:        j,
			AnnSensus: as,
		}
		var peers []int
		for k := 0; k < 4; k++ {
			if k == j {
				continue
			}
			peers = append(peers, k)
		}
		pub, priv, _ := crypto.NewSigner(crypto.CryptoTypeSecp256k1).RandomKeyPair()
		as.MyAccount = &account.SampleAccount{
			PrivateKey: priv,
			PublicKey:  pub,
			Address:    pub.Address(),
			Id:         j,
		}
		as.Idag = &DummyDag{}
		a.Hub = newtestHub(j, peers, sendMsgToChan, sendMsgByPubKey, as)
		a.AnnSensus = as
		//logrus.WithField("addr ", a.MyPrivKey.PublicKey().Address().TerminalString()).Debug("gen hub ", a.Hub)
		Anns = append(Anns, a)
	}

	//start here
	//collect campaigns
	og.MsgCountInit()
	var cps types.Txis
	for _, ann := range Anns {
		cps = append(cps, ann.GenCampaign())
	}
	logrus.Debug("gen camp ", cps)
	for _, ann := range Anns {
		ann.Start()
	}
	time.Sleep(20 * time.Millisecond)
	for _, ann := range Anns {
		ann.ConsensusTXConfirmed <- cps
	}

	time.Sleep(time.Second * 4)
	for _, ann := range Anns {
		ann.Stop()
	}

}

func (as *TestAnnSensus) Start() {
	as.campaignFlag = false
	as.AnnSensus.Start()
	go as.Hub.(*TestHub).loop()
	logrus.Info("started ann  ", as.Id)
}

func (a *TestAnnSensus) Stop() {
	a.AnnSensus.Stop()
	a.Hub.(*TestHub).quit <- struct{}{}
	logrus.Info("stopped ann ", a.Id)
}

func (as *TestAnnSensus) GenCampaign() *types.Campaign {
	// generate campaign.
	camp := as.genCamp(as.dkg.pk)
	as.newCampaign(camp)
	ok := as.VerifyCampaign(camp)
	if !ok {
		panic(ok)
	}
	return camp
}

func (as *TestAnnSensus) newCampaign(cp *types.Campaign) {
	cp.GetBase().PublicKey = as.MyAccount.PublicKey.Bytes
	cp.GetBase().AccountNonce = uint64(as.Id * 10)
	cp.Issuer = as.MyAccount.Address
	s := crypto.NewSigner(as.cryptoType)
	cp.GetBase().Signature = s.Sign(as.MyAccount.PrivateKey, cp.SignatureTargets()).Bytes
	cp.GetBase().Weight = uint64(as.Id * 100)
	cp.Height = uint64(as.Id)
	cp.GetBase().Hash = cp.CalcTxHash()
	return
}

func logInit() {

	Formatter := new(logrus.TextFormatter)
	//Formatter.DisableColors = true
	Formatter.TimestampFormat = "15:04:05.000000"
	Formatter.FullTimestamp = true
	logrus.SetFormatter(Formatter)
	logrus.SetLevel(logrus.TraceLevel)
	filenameHook := filename.NewHook()
	filenameHook.Field = "line"
	logrus.AddHook(filenameHook)
}
