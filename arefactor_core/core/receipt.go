// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package core

import (
	"fmt"
	"github.com/annchain/OG/common"
)

type ReceiptStatus uint8

const (
	ReceiptStatusSuccess ReceiptStatus = iota
	ReceiptStatusOVMFailed
	ReceiptStatusUnknownTxType
	ReceiptStatusFailed
)

//go:generate msgp

//msgp:tuple Receipt
type Receipt struct {
	TxHash          common.Hash
	Status          ReceiptStatus
	ProcessResult   string
	ContractAddress common.Address
}

func NewReceipt(hash common.Hash, status ReceiptStatus, pResult string, addr common.Address) *Receipt {
	return &Receipt{
		TxHash:          hash,
		Status:          status,
		ProcessResult:   pResult,
		ContractAddress: addr,
	}
}

func (r *Receipt) ToJsonMap() map[string]interface{} {
	jm := make(map[string]interface{})
	jm["hash"] = r.TxHash.Hex()
	jm["status"] = fmt.Sprintf("%d", r.Status)
	jm["result"] = r.ProcessResult
	jm["contractAddress"] = r.ContractAddress.Hex()

	return jm
}

//msgp:tuple ReceiptSet
type ReceiptSet map[string]*Receipt
