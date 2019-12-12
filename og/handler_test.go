package og

import (
	"fmt"
	"github.com/annchain/OG/og/types"
	"github.com/annchain/OG/og/types/archive"
	"testing"
)

func TestIncomingMessageHandler_HandleBodiesRequest(t *testing.T) {
	var msgRes archive.MessageBodiesResponse
	var bytes int
	for i := 0; i < 2; i++ {
		seq := types.RandomSequencer()

		if bytes >= softResponseLimit {
			message_archive.msgLog.Debug("reached softResponseLimit")
			break
		}
		if len(msgRes.Bodies) >= 400000 {
			message_archive.msgLog.Debug("reached MaxBlockFetch 128")
			break
		}
		var body types.MessageBodyData
		body.RawSequencer = seq.RawSequencer()
		var txs types.Txis
		for j := 0; j < 3; j++ {
			txs = append(txs, archive.RandomTx())
		}
		rtxs := types.NewTxisMarshaler(txs)
		if rtxs != nil && len(rtxs) != 0 {
			body.RawTxs = &rtxs
		}
		bodyData, _ := body.MarshalMsg(nil)
		bytes += len(bodyData)
		msgRes.Bodies = append(msgRes.Bodies, archive.RawData(bodyData))
		fmt.Println(body)
	}
	fmt.Println(&msgRes)
}
