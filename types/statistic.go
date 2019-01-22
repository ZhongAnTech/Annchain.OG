package types

//go:generate msgp
type ConfirmTime struct {
	SeqHeight   uint64 `json:"seq_Height"`
	TxNum       uint64 `json:"tx_num"`
	ConfirmTime string `json:"confirm_time"`
}
