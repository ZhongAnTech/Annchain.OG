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
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/gcache"
	"github.com/annchain/OG/consensus/annsensus/test"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"github.com/sirupsen/logrus"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"math/rand"
	"testing"
	"time"
)

type TestAnnSensus struct {
	*AnnSensus
}

type AId struct {
	Address common.Address
	dkgId   int
}

func (a AId) String() string {
	return fmt.Sprintf("add %s  dkgId %d", a.Address.TerminalString(), a.dkgId)
}

func (a *TestAnnSensus) Aid() AId {
	return AId{
		Address: a.MyAccount.Address,
		dkgId:   int(a.dkg.GetId()),
	}
}

func GetAnn(anns []TestAnnSensus, Addr common.Address) *TestAnnSensus {
	for i, ann := range anns {
		if bytes.Equal(ann.MyAccount.Address.ToBytes(), Addr.ToBytes()) {
			return &anns[i]
		}
	}
	panic("not found")
}

type p2pMsg struct {
	data    []byte
	msgType og.MessageType
}

type TestHub struct {
	Id              common.Address
	Peers           []common.Address
	sendMsgToChan   sendMsgToChanFunc
	sendMsgByPubKey sendMsgByPubKeyFunc
	OutMsg          chan p2pMsg
	quit            chan struct{}
	As              *TestAnnSensus
	msgCache        gcache.Cache
}

type sendMsgToChanFunc func(addr common.Address, mdg test.TestMsg)
type sendMsgByPubKeyFunc func(pub *crypto.PublicKey, msg test.TestMsg)

func newtestHub(id common.Address, peers []common.Address, sendMsgToChan sendMsgToChanFunc, sendMsgByPubKey sendMsgByPubKeyFunc, as *TestAnnSensus) *TestHub {
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

func (t *TestHub) BroadcastMessage(messageType og.MessageType, message p2p_message.Message) {
	var sent bool
	for _, peer := range t.Peers {
		tMsg := test.TestMsg{
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

func (t *TestHub) SendToPeer(peerId string, messageType og.MessageType, msg p2p_message.Message) error {
	return nil
}

func (t *TestHub) SendToAnynomous(messageType og.MessageType, message p2p_message.Message, anyNomousPubKey *crypto.PublicKey) {
	tMsg := test.TestMsg{
		MessageType: messageType,
		Message:     message,
		From:        t.Id,
	}
	t.sendMsgByPubKey(anyNomousPubKey, tMsg)
	hash := tMsg.GetHash()
	t.msgCache.Set(hash, struct{}{})
}

func (as *TestAnnSensus) Start() {
	as.campaignFlag = false
	as.dkg.Start()
	go as.loop()
	go as.Hub.(*TestHub).loop()
	logrus.Info("started ann  ", as.dkg.GetId())
}

func (a *TestAnnSensus) Stop() {
	a.dkg.Stop()
	close(a.close)
	a.Hub.(*TestHub).quit <- struct{}{}
	logrus.Info("stopped ann ", a.dkg.GetId())
}

func (as *TestAnnSensus) GenCampaign() *tx_types.Campaign {
	// generate campaign.
	//generate new dkg public key for every campaign
	candidatePublicKey := as.dkg.GenerateDkg()
	// generate campaign.
	camp := as.genCamp(candidatePublicKey)

	as.newCampaign(camp)
	err := camp.UnmarshalDkgKey(bn256.UnmarshalBinaryPointG2)
	if err != nil {
		panic(err)
	}
	//ok := as.VerifyCampaign(camp)
	//if !ok {
	//	panic(ok)
	//}
	return camp
}

func (as *TestAnnSensus) newCampaign(cp *tx_types.Campaign) {
	cp.GetBase().PublicKey = as.MyAccount.PublicKey.Bytes
	cp.GetBase().AccountNonce = uint64(as.dkg.GetId() * 10)
	cp.Issuer = &as.MyAccount.Address
	s := crypto.NewSigner(as.cryptoType)
	cp.GetBase().Signature = s.Sign(as.MyAccount.PrivateKey, cp.SignatureTargets()).Bytes
	cp.GetBase().Weight = uint64(rand.Int31n(100)%10 + 3)
	cp.Height = uint64(as.dkg.TermId + 3)
	cp.GetBase().Hash = cp.CalcTxHash()
	return
}

func (t *TestHub) loop() {
	elog := logrus.WithField("me", t.Id).WithField("aid ", t.As.Aid().String())
	for {
		select {
		case pMsg := <-t.OutMsg:
			var msg test.TestMsg
			switch pMsg.msgType {
			case og.MessageTypeConsensusDkgDeal:
				msg.MessageType = pMsg.msgType
				msg.Message = &p2p_message.MessageConsensusDkgDeal{}
			case og.MessageTypeConsensusDkgDealResponse:
				msg.MessageType = pMsg.msgType
				msg.Message = &p2p_message.MessageConsensusDkgDealResponse{}
			case og.MessageTypeConsensusDkgSigSets:
				msg.MessageType = pMsg.msgType
				msg.Message = &p2p_message.MessageConsensusDkgSigSets{}
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
				request := msg.Message.(*p2p_message.MessageConsensusDkgDeal)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.Data)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate dkg msg")
					continue
				}
				go t.As.HandleConsensusDkgDeal(request, fmt.Sprintf("%s", msg.From.TerminalString()))
			case og.MessageTypeConsensusDkgDealResponse:
				request := msg.Message.(*p2p_message.MessageConsensusDkgDealResponse)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.Data)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate response  msg")
					continue
				}
				go t.As.HandleConsensusDkgDealResponse(request, fmt.Sprintf("%s", msg.From.TerminalString()))
			case og.MessageTypeConsensusDkgSigSets:
				request := msg.Message.(*p2p_message.MessageConsensusDkgSigSets)
				if _, err := t.msgCache.GetIFPresent(hash); err == nil {
					//elog.WithField("from ", msg.From).WithField("msg type",
					//	msg.MessageType).WithField("msg ", len(request.PkBls)).WithField("hash ",
					//	msg.GetHash()).Warn("duplicate response  msg")
					continue
				}
				go t.As.HandleConsensusDkgSigSets(request, fmt.Sprintf("%s", msg.From.TerminalString()))
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

func run(n int) {
	logInit()
	var Anns []TestAnnSensus
	sendMsgToChan := func(addr common.Address, msg test.TestMsg) {
		data, err := msg.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		pMsg := p2pMsg{data: data, msgType: msg.MessageType}
		ann := GetAnn(Anns, addr)
		ann.Hub.(*TestHub).OutMsg <- pMsg
		var resp *p2p_message.MessageConsensusDkgDealResponse
		if msg.MessageType == og.MessageTypeConsensusDkgDealResponse {
			resp = msg.Message.(*p2p_message.MessageConsensusDkgDealResponse)
		}
		logrus.WithField("me ", msg.From).WithField("to peer ", ann.Aid().String()).WithField("type ",
			msg.MessageType).WithField("msg ", resp).WithField("len ", len(data)).Trace("send msg")
		return
	}
	sendMsgByPubKey := func(pub *crypto.PublicKey, msg test.TestMsg) {
		data, err := msg.MarshalMsg(nil)
		if err != nil {
			panic(err)
		}
		pMsg := p2pMsg{data: data, msgType: msg.MessageType}
		for j := 0; j < n; j++ {
			if bytes.Equal(Anns[j].MyAccount.PublicKey.Bytes, pub.Bytes) {
				Anns[j].Hub.(*TestHub).OutMsg <- pMsg
				logrus.WithField("from peer", msg.From.TerminalString()).WithField("to peer id ", Anns[j].Aid().String()).WithField("type ",
					msg.MessageType).Trace("send msg enc")
				return
			}
		}
		logrus.Warn("not found for pubkey ", pub)
		return
	}

	var accounts []account.SampleAccount
	var pks []crypto.PublicKey
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	for j := 0; j < n; j++ {
		pub, priv := signer.RandomKeyPair()
		account := account.SampleAccount{
			PrivateKey: priv,
			PublicKey:  pub,
			Address:    pub.Address(),
			Id:         j,
		}
		accounts = append(accounts, account)
		pks = append(pks, account.PublicKey)
	}

	for j := 0; j < n; j++ {
		as := NewAnnSensus(4, false, crypto.CryptoTypeSecp256k1, true, n,
			pks, "test.json", false)
		a := TestAnnSensus{
			AnnSensus: as,
		}
		var peers []common.Address
		for k := 0; k < n; k++ {
			if k == j {
				continue
			}
			peers = append(peers, accounts[k].Address)
		}
		hub := newtestHub(accounts[j].Address, peers, sendMsgToChan, sendMsgByPubKey, &a)
		as.InitAccount(&accounts[j], time.Second, nil, nil, &DummyDag{}, nil, nil, hub)
		a.AnnSensus = as
		logrus.WithField("addr ", a.MyAccount.Address.TerminalString()).Debug("gen hub done")
		Anns = append(Anns, a)
	}

	//start here
	//collect campaigns
	og.MsgCountInit()
	for i := range Anns {
		Anns[i].Start()
	}
	time.Sleep(20 * time.Millisecond)
	var num int
	for {
		select {
		case <-time.After(time.Second * 5):
			num++
			if num > 10 {
				return
			}
			var cps types.Txis
			for _, ann := range Anns {
				cps = append(cps, ann.GenCampaign())
			}
			logrus.Debug("gen camp ", cps)
			for i := range Anns {
				go func(i int) {
					log.WithField("i ", i).WithField("ann ", Anns[i].dkg.GetId()).Debug("new term start")
					Anns[i].ConsensusTXConfirmed <- cps
				}(i)
			}
		case <-time.After(time.Second * 21):
			break
		}

	}

	for i := range Anns {
		Anns[i].Stop()
	}

}

func (as *TestAnnSensus) loop() {
	for {
		select {
		case tc := <-as.termChangeChan:
			log.WithField("me ", as.dkg.GetId()).WithField("pk ", tc).Info("got term change and pk")
			as.term.ChangeTerm(tc, as.Idag.GetHeight())
			//
		case <-as.newTermChan:
			// sequencer generate entry.

			// TODO:
			// 1. check if yourself is the miner.
			// 2. produce raw_seq and broadcast it to network.
			// 3. start bft until someone produce a seq with BLS sig.
			log.Info("got newTermChange signal")
		case <-as.startTermChange:
			log := as.dkg.Log()
			cp := as.term.GetCampaign(as.MyAccount.Address)
			log.WithField("cp ", cp).Debug("will reset with cp")
			as.dkg.Reset(cp)
			as.dkg.SelectCandidates(as.Idag.LatestSequencer())
			if !as.dkg.IsValidPartner() {
				log.Debug("i am not a lucky dkg partner quit")

				continue
			}
			log.Debug("start dkg gossip")
			go as.dkg.StartGossip()

		case pk := <-as.dkgPulicKeyChan:
			log := as.dkg.Log()
			log.WithField("pk ", pk).Info("got a bls public key from dkg")
			//after dkg  gossip finished, set term change flag
			//term change is finished
			as.term.SwitchFlag(false)
			sigset := as.dkg.GetBlsSigsets()
			log.WithField("sig sets ", sigset).Info("got sigsets ")
			//continue //for test case commit this
			tc := as.genTermChg(pk, sigset)
			if tc == nil {
				log.Warn("tc is nil")
				continue
			}
			go func() {
				as.termChangeChan <- tc
			}()

		case txs := <-as.ConsensusTXConfirmed:
			var cps tx_types.Campaigns
			for _, tx := range txs {
				if tx.GetType() == types.TxBaseTypeCampaign {
					cp := tx.(*tx_types.Campaign)
					cps = append(cps, cp)
					if bytes.Equal(cp.Issuer.Bytes[:], as.MyAccount.Address.Bytes[:]) {

					}
				}
			}
			as.newTerm(cps)

		case <-as.close:
			log.Info("got quit signal , annsensus loop stopped")
			return
		}
	}

}

func (as *TestAnnSensus) newTerm(cps tx_types.Campaigns) {
	log.Trace("new term change")
	if len(cps) > 0 {
		for _, c := range cps {
			err := as.AddCampaign(c)
			if err != nil {
				log.WithError(err).Debug("add campaign err")
				//continue
			}
		}
	}
	// start term changing.
	as.term.SwitchFlag(true)
	log.Debug("will term Change")
	go func() {
		as.startTermChange <- true
	}()

	log.Infof("already candidates: %d, alsorans: %d", len(as.Candidates()), len(as.Alsorans()))
}

func TestDKGMain(t *testing.T) {
	run(2)
}
