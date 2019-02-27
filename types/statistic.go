package types

//go:generate msgp
//msgp:tuple ConfirmTime
type ConfirmTime struct {
	SeqHeight   uint64 `json:"seq_Height"`
	TxNum       uint64 `json:"tx_num"`
	ConfirmTime string `json:"confirm_time"`
}
