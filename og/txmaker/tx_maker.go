package txmaker

import "github.com/annchain/OG/og/protocol/ogmessage"

// TxMaker makes new Tx (or Seq) using the infomation that InfoProvider gives.
type TxMaker interface {
	MakeTx(txType ogmessage.TxBaseType) ogmessage.Txi
}
