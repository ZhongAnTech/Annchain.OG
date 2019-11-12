package txmaker

import (
	"github.com/annchain/OG/og/protocol/ogmessage"
	"github.com/annchain/OG/og/protocol/ogmessage/archive"
)

// TxMaker makes new Tx (or Seq) using the infomation that InfoProvider gives.
type TxMaker interface {
	MakeTx(txType archive.TxBaseType) ogmessage.Txi
}
