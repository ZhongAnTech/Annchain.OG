package syncer

import (
	"fmt"
	"github.com/annchain/OG/og"
	"github.com/annchain/OG/types"
	"sync"
	"time"
)

//adding bloom filter status , avoid send too frequently, after sending a request , wait until got response or time out
//
type BloomFilterFireStatus struct  {
	mu sync.RWMutex
	startTime  time.Time
	requestId  uint32
	gotResponse  bool
	minFrequency  time.Duration
	responseTimeOut time.Duration
}

func NewBloomFilterFireStatus (minFrequencyTime int, responseTimeOut int  )*BloomFilterFireStatus{
	if minFrequencyTime >=responseTimeOut{
		panic(fmt.Sprintf("param err %v, %v",minFrequencyTime,responseTimeOut))
	}
	return &BloomFilterFireStatus{
		requestId:0,
		gotResponse:true,
		minFrequency:time.Duration(minFrequencyTime)*time.Millisecond,
		responseTimeOut:time.Millisecond*time.Duration(responseTimeOut),
	}
}

func (b*BloomFilterFireStatus)set(requestId uint32) {
	b.requestId = requestId
	b.gotResponse = false
	b.startTime = time.Now()
}

func (b*BloomFilterFireStatus)Set(requestId uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.set(requestId)
}

func (b*BloomFilterFireStatus)UpdateResponse(requestId uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.updateResponse(requestId)
}

func (b*BloomFilterFireStatus)updateResponse(requestId uint32) {
	if b.requestId !=requestId {
		return
	}
	b.gotResponse = true
	log.WithField("requestId ",requestId).WithField("after ",time.Now().Sub(b.startTime)).Debug(
		"bloom filter got response after")
	return
}

func (b*BloomFilterFireStatus)Check() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.check()
}

func (b*BloomFilterFireStatus)check() bool {
	stamp := time.Now().Sub(b.startTime)
	if stamp > b.responseTimeOut  {
		return true
	}
	if b.gotResponse == true && stamp >  b.minFrequency{
		return  true
	}
	return false
}


//sendBloomFilter , avoid sending blomm filter frequently ,wait until got response of bloom filter or timeout

func (m*IncrementalSyncer)sendBloomFilter(hash types.Hash) {
	m.bloomFilterStatus.mu.Lock()
	if !m.bloomFilterStatus.check() {
		log.Debug("bloom filter request is pending")
		return
	}
	requestId := og.MsgCounter.Get()
	m.bloomFilterStatus.set(requestId)
	m.bloomFilterStatus.mu.Unlock()
	req := types.MessageSyncRequest{
		Filter:    types.NewDefaultBloomFilter(),
		RequestId: og.MsgCounter.Get(),
	}
	m.bloomFilterStatus.Set(req.RequestId)
	height := m.getHeight()
	req.Height = &height
	hashs := m.getTxsHashes()
	for _, hash := range hashs {
		req.Filter.AddItem(hash.Bytes[:])
	}
	err := req.Filter.Encode()
	if err != nil {
		log.WithError(err).Warn("encode filter err")
	}
	log.WithField("height ", height).WithField("type", og.MessageTypeFetchByHashRequest).WithField(
		"req ",req.String()).WithField("filter length", len(req.Filter.Data)).Debug(
		"sending bloom filter  MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(og.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	if len(hashs) == 0 {
		//source unknown
		m.messageSender.MulticastMessage(og.MessageTypeFetchByHashRequest, &req)
		return
	}
	var sourceHash *types.Hash
	sourceHash = &hash
	m.messageSender.MulticastToSource(og.MessageTypeFetchByHashRequest, &req, sourceHash)
	return
}
