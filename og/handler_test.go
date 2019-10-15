package og

import (
	"fmt"
	"github.com/annchain/OG/og/message"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/types/p2p_message"
	"testing"
)

func TestIncomingMessageHandler_HandleBodiesRequest(t *testing.T) {
	var msgRes p2p_message.MessageBodiesResponse
	var bytes int
	for i := 0; i < 2; i++ {
		seq := protocol_message.RandomSequencer()

		if bytes >= softResponseLimit {
			message.msgLog.Debug("reached softResponseLimit")
			break
		}
		if len(msgRes.Bodies) >= 400000 {
			message.msgLog.Debug("reached MaxBlockFetch 128")
			break
		}
		var body p2p_message.MessageBodyData
		body.RawSequencer = seq.RawSequencer()
		var txs protocol_message.Txis
		for j := 0; j < 3; j++ {
			txs = append(txs, protocol_message.RandomTx())
		}
		rtxs := protocol_message.NewTxisMarshaler(txs)
		if rtxs != nil && len(rtxs) != 0 {
			body.RawTxs = &rtxs
		}
		bodyData, _ := body.MarshalMsg(nil)
		bytes += len(bodyData)
		msgRes.Bodies = append(msgRes.Bodies, p2p_message.RawData(bodyData))
		fmt.Println(body)
	}
	fmt.Println(&msgRes)
}
