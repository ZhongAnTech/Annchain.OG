package core

import (
	"github.com/annchain/OG/types"
)

type ReceiptStatus int

const (
	ReceiptStatusSeqSuccess ReceiptStatus = iota
	ReceiptStatusTxSuccess
	ReceiptStatusOVMFailed
)

//go:generate msgp

//msgp:tuple Receipt
type Receipt struct {
	TxHash          types.Hash
	Status          ReceiptStatus
	ProcessResult   string
	ContractAddress types.Address
}

func NewReceipt(hash types.Hash, status ReceiptStatus, pResult string, addr types.Address) *Receipt {
	return &Receipt{
		TxHash:          hash,
		Status:          status,
		ProcessResult:   pResult,
		ContractAddress: addr,
	}
}

//msgp:tuple ReceiptSet
type ReceiptSet map[string]*Receipt
