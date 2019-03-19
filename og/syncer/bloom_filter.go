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
type BloomFilterFireStatus struct {
	mu              sync.RWMutex
	startTime       time.Time
	requestId       uint32
	gotResponse     bool
	minFrequency    time.Duration
	responseTimeOut time.Duration
}

func NewBloomFilterFireStatus(minFrequencyTime int, responseTimeOut int) *BloomFilterFireStatus {
	if minFrequencyTime >= responseTimeOut {
		panic(fmt.Sprintf("param err %v, %v", minFrequencyTime, responseTimeOut))
	}
	return &BloomFilterFireStatus{
		requestId:       0,
		gotResponse:     true,
		minFrequency:    time.Duration(minFrequencyTime) * time.Millisecond,
		responseTimeOut: time.Millisecond * time.Duration(responseTimeOut),
	}
}

func (b *BloomFilterFireStatus) set(requestId uint32) {
	b.requestId = requestId
	b.gotResponse = false
	b.startTime = time.Now()
}

func (b *BloomFilterFireStatus) Set(requestId uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.set(requestId)
}

func (b *BloomFilterFireStatus) UpdateResponse(requestId uint32) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	b.updateResponse(requestId)
}

func (b *BloomFilterFireStatus) updateResponse(requestId uint32) {
	if b.requestId != requestId {
		return
	}
	b.gotResponse = true
	log.WithField("requestId ", requestId).WithField("after ", time.Now().Sub(b.startTime)).Debug(
		"bloom filter got response after")
	return
}

func (b *BloomFilterFireStatus) Check() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.check()
}

func (b *BloomFilterFireStatus) check() bool {
	stamp := time.Now().Sub(b.startTime)
	if stamp > b.responseTimeOut {
		return true
	}
	if b.gotResponse == true && stamp > b.minFrequency {
		return true
	}
	return false
}

//sendBloomFilter , avoid sending bloom filter frequently ,wait until got response of bloom filter or timeout

func (m *IncrementalSyncer) sendBloomFilter(childhash types.Hash) {
	if !m.bloomFilterStatus.Check() {
		log.Debug("bloom filter request is pending")
		return
	}
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
		"req ", req.String()).WithField("filter length", len(req.Filter.Data)).Debug(
		"sending bloom filter  MessageTypeFetchByHashRequest")

	//m.messageSender.UnicastMessageRandomly(og.MessageTypeFetchByHashRequest, bytes)
	//if the random peer dose't have this txs ,we will get nil response ,so broadcast it
	hash := childhash
	m.messageSender.MulticastToSource(og.MessageTypeFetchByHashRequest, &req, &hash)
	return
}
