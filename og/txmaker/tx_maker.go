package txmaker

import "github.com/annchain/OG/types"

// TxMaker makes new Tx (or Seq) using the infomation that InfoProvider gives.
type TxMaker interface {
	MakeTx(txType types.TxBaseType) types.Txi
}
