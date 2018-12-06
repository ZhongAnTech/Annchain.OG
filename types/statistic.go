package types

//go:generate msgp
type ConfirmTime struct {
	SeqId        uint64 `json:"seq_id"`
	TxNum        uint64 `json:"tx_num"`
	ConfirmTime  string  `json:"confirm_time"`
}
