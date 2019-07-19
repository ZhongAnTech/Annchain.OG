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
package rpc

import (
	"fmt"
	"github.com/annchain/OG/p2p"
	"github.com/annchain/OG/p2p/ioperformance"
	"github.com/annchain/OG/types"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

type SyncStatus struct {
	Id                       string `json:"id"`
	SyncMode                 string `json:"syncMode"`
	CatchupSyncerStatus      string `json:"catchupSyncerStatus"`
	CatchupSyncerEnabled     bool   `json:"catchupSyncerEnabled"`
	IncrementalSyncerEnabled bool   `json:"incrementalSyncerEnabled"`
	Height                   uint64 `json:"height"`
	LatestHeight             uint64 `json:"latestHeight"`
	BestPeer                 string `json:"bestPeer"`
	Error                    string `json:"error"`
	Txid                     uint32 `json:"txid"`
}

//NodeStatus
type NodeStatus struct {
	NodeInfo  *p2p.NodeInfo   `json:"node_info"`
	PeersInfo []*p2p.PeerInfo `json:"peers_info"`
}

//Status node status
func (r *RpcController) SyncStatus(c *gin.Context) {
	status := r.syncStatus()
	cors(c)
	c.JSON(http.StatusOK, status)
}

func (r *RpcController) syncStatus() SyncStatus {
	var status SyncStatus

	status = SyncStatus{
		Id:                       r.P2pServer.Self().ID().TerminalString(),
		SyncMode:                 r.SyncerManager.Status.String(),
		CatchupSyncerStatus:      r.SyncerManager.CatchupSyncer.WorkState.String(),
		CatchupSyncerEnabled:     r.SyncerManager.CatchupSyncer.Enabled,
		IncrementalSyncerEnabled: r.SyncerManager.IncrementalSyncer.Enabled,
		Height:                   r.SyncerManager.NodeStatusDataProvider.GetCurrentNodeStatus().CurrentId,
	}

	peerId, _, seqId, err := r.SyncerManager.Hub.BestPeerInfo()
	if err != nil {
		status.Error = err.Error()
	} else {
		status.LatestHeight = seqId
		status.BestPeer = peerId

	}
	status.Txid = r.Og.TxPool.GetTxNum()
	return status
}

func (r *RpcController) Performance(c *gin.Context) {
	cd := r.PerformanceMonitor.CollectData()
	cors(c)
	c.JSON(http.StatusOK, cd)
}

func (r *RpcController) NetIo(c *gin.Context) {
	cors(c)
	type transportData struct {
		*ioperformance.IoDataInfo `json:"transport_data"`
	}
	data := transportData{ioperformance.GetNetPerformance()}
	Response(c, http.StatusOK, nil, data)
	return
}

func (r *RpcController) OgPeersInfo(c *gin.Context) {
	info := r.Og.Manager.Hub.PeersInfo()

	Response(c, http.StatusOK, nil, info)
	return
}

type Monitor struct {
	Port    string     `json:"port"`
	ShortId string     `json:"short_id"`
	Peers   []Peer     `json:"peers,omitempty"`
	SeqId   uint64     `json:"seq_id"`
	Tps     *Tps       `json:"tps"`
	Status  SyncStatus `json:"status"`
}

type Peer struct {
	Addr    string `json:"addr"`
	ShortId string `json:"short_id"`
	Link    bool   `json:"link"`
}

func (r *RpcController) Monitor(c *gin.Context) {
	var m Monitor
	seq := r.Og.Dag.LatestSequencer()
	if seq != nil {
		m.SeqId = seq.Height
	}
	peersinfo := r.Og.Manager.Hub.PeersInfo()
	for _, p := range peersinfo {
		/*
			if p.Network.Inbound {
				addr = p.Network.LocalAddress
			}else {
				addr = p.Network.RemoteAddress
			}
				ipPort :=strings.Split(addr,":")
				if len(ipPort) ==2 {
					m.Peers = append(m.Peers ,ipPort[1])
				}
		*/
		var peer Peer
		peer.Addr = p.Addrs
		peer.ShortId = p.ShortId
		peer.Link = p.Link
		m.Peers = append(m.Peers, peer)
	}
	m.Port = viper.GetString("p2p.port")
	m.ShortId = r.P2pServer.NodeInfo().ShortId
	m.Tps, _ = r.getTps()
	m.Status = r.syncStatus()

	Response(c, http.StatusOK, nil, m)
	return
}

func (r *RpcController) BftStatus(c *gin.Context) {
	cors(c)
	if r.AnnSensus != nil {
		s := r.AnnSensus.GetBftStatus()
		Response(c, http.StatusOK, nil, s)
	} else {
		Response(c, http.StatusOK, nil, nil)
	}

	return
}

func (r *RpcController) GetPoolHashes(c *gin.Context) {
	cors(c)
	s := r.Og.TxPool.GetOrder()
	Response(c, http.StatusOK, nil, s)
	return
}

func (r *RpcController) Validator(c *gin.Context) {
	cors(c)
	Response(c, http.StatusOK, nil, "validator")
	return
}

type Tps struct {
	Num     int     `json:"num"`
	TxCount int     `json:"tx_num"`
	Seconds float64 `json:"duration"`
}

func (r *RpcController) getTps() (t *Tps, err error) {
	var tps Tps
	lseq := r.Og.Dag.LatestSequencer()
	if lseq == nil {
		return nil, fmt.Errorf("not found")
	}
	if lseq.Height < 3 {
		return
	}

	var cfs []types.ConfirmTime
	for id := lseq.Height; id > 0 && id > lseq.Height-5; id-- {
		cf := r.Og.Dag.GetConfirmTime(id)
		if cf == nil {
			return nil, fmt.Errorf("db error")
		}
		cfs = append(cfs, *cf)
	}
	var start, end time.Time
	for i, cf := range cfs {
		if i == 0 {
			end, err = time.Parse(time.RFC3339Nano, cf.ConfirmTime)
			if err != nil {
				return nil, err
			}
		}
		if i == len(cfs)-1 {
			start, err = time.Parse(time.RFC3339Nano, cf.ConfirmTime)
			if err != nil {
				return nil, err
			}
		} else {
			tps.TxCount += int(cf.TxNum)
		}
	}

	if !end.After(start) {
		return nil, fmt.Errorf("time server error")
	}
	sub := end.Sub(start)
	sec := sub.Seconds()
	if sec != 0 {
		num := float64(tps.TxCount) / sec
		tps.Num = int(num)
	}
	tps.Seconds = sec
	return &tps, nil
}

func (r *RpcController) Tps(c *gin.Context) {
	cors(c)
	t, err := r.getTps()
	if err != nil {
		Response(c, http.StatusBadRequest, err, nil)
		return
	}
	Response(c, http.StatusOK, nil, t)
	return
}

//Status node status
func (r *RpcController) Status(c *gin.Context) {
	var status NodeStatus
	status.NodeInfo = r.P2pServer.NodeInfo()
	status.PeersInfo = r.P2pServer.PeersInfo()
	cors(c)
	Response(c, http.StatusOK, nil, status)
}

//PeersInfo network information
func (r *RpcController) NetInfo(c *gin.Context) {
	info := r.P2pServer.NodeInfo()
	cors(c)
	Response(c, http.StatusOK, nil, info)
}

//PeersInfo  peers information
func (r *RpcController) PeersInfo(c *gin.Context) {
	peersInfo := r.P2pServer.PeersInfo()
	cors(c)
	Response(c, http.StatusOK, nil, peersInfo)
}
