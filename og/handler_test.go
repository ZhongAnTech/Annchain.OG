package og

import (
	"fmt"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
	"testing"
)

func TestIncomingMessageHandler_HandleBodiesRequest(t *testing.T) {
	var msgRes archive.MessageBodiesResponse
	var bytes int
	for i := 0; i < 2; i++ {
		seq := ogmessage.RandomSequencer()

		if bytes >= softResponseLimit {
			message.msgLog.Debug("reached softResponseLimit")
			break
		}
		if len(msgRes.Bodies) >= 400000 {
			message.msgLog.Debug("reached MaxBlockFetch 128")
			break
		}
		var body ogmessage.MessageBodyData
		body.RawSequencer = seq.RawSequencer()
		var txs ogmessage.Txis
		for j := 0; j < 3; j++ {
			txs = append(txs, archive.RandomTx())
		}
		rtxs := ogmessage.NewTxisMarshaler(txs)
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
