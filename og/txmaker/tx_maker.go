package txmaker

import (
	"github.com/annchain/OG/og/protocol_message"
)

// TxMaker makes new Tx (or Seq) using the infomation that InfoProvider gives.
type TxMaker interface {
	MakeTx(txType protocol_message.TxBaseType) protocol_message.Txi
}
