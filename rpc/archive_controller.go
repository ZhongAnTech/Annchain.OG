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
	"bytes"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/annchain/OG/types"
	"github.com/annchain/OG/vm/eth/common/math"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var archiveId uint32

func getArchiveId() uint32 {
	if archiveId > math.MaxUint32-1000 {
		archiveId = 10
	}
	return atomic.AddUint32(&archiveId, 1)
}

// NewArchive 新建Archive交易
func (r *RpcController) NewArchive(c *gin.Context) {
	var (
		tx    types.Txi
		txReq NewArchiveRequest
	)
	now := time.Now()
	id := getArchiveId()
	//if !status.ArchiveMode {
	//	Response(c, http.StatusBadRequest, fmt.Errorf("not archive mode"), nil)
	//	return
	//}
	err := c.ShouldBindJSON(&txReq)
	if err != nil {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: %v", err), nil)
		return
	}
	// 处理NewArchiveRequest类型数据
	// 验证签名和存证哈希
	if !txReq.Verify() {
		fmt.Println("New archive not verified.")
		return
	}
	//c.Request.Context()
	if len(txReq.Data) == 0 {
		Response(c, http.StatusBadRequest, fmt.Errorf("request format error: no data "), nil)
		return
	}
	var buf bytes.Buffer
	buf.Write(txReq.Data)
	//TODO compress data
	logrus.WithField("id ", id).WithField("data  ", string(txReq.Data)).Trace("got archive request")
	tx, err = r.TxCreator.NewArchiveWithSeal(buf.Bytes())
	if err != nil {
		Response(c, http.StatusInternalServerError, fmt.Errorf("new tx failed %v", err), nil)
		return
	}
	logrus.WithField("id ", id).WithField("tx", tx).Debugf("tx generated")
	if !r.SyncerManager.IncrementalSyncer.Enabled {
		Response(c, http.StatusOK, fmt.Errorf("tx is disabled when syncing"), nil)
		return
	}
	//TODO which one is fast
	//r.SyncerManager.IncrementalSyncer.CacheTx(tx)

	r.TxBuffer.ReceivedNewTxChan <- tx
	logrus.WithField("used time ", time.Since(now)).WithField("id ", id).WithField("tx ", tx).Trace("send ok")

	// 返回交易哈希
	Response(c, http.StatusOK, nil, tx.GetTxHash().Hex())
	return
}
