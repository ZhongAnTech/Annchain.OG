package core

import (
	"fmt"

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

func (r *Receipt) ToJsonMap() map[string]string {
	jm := make(map[string]string)
	jm["hash"] = r.TxHash.Hex()
	jm["status"] = fmt.Sprintf("%d", r.Status)
	jm["result"] = r.ProcessResult
	jm["contractAddress"] = r.ContractAddress.Hex()

	return jm
}

//msgp:tuple ReceiptSet
type ReceiptSet map[string]*Receipt
