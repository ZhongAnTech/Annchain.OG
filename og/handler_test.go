package og

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/p2p_message"
	"github.com/annchain/OG/types/tx_types"
	"testing"
)

func TestIncomingMessageHandler_HandleBodiesRequest(t *testing.T) {
	var msgRes p2p_message.MessageBodiesResponse
	var bytes int
	for i := 0; i < 2; i++ {
		seq := tx_types.RandomSequencer()

		if bytes >= softResponseLimit {
			msgLog.Debug("reached softResponseLimit")
			break
		}
		if len(msgRes.Bodies) >= 400000 {
			msgLog.Debug("reached MaxBlockFetch 128")
			break
		}
		var body p2p_message.MessageBodyData
		body.RawSequencer = seq.RawSequencer()
		var txs types.Txis
		for j := 0; j < 3; j++ {
			txs = append(txs, tx_types.RandomTx())
		}
		rtxs := tx_types.NewTxisMarshaler(txs)
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
