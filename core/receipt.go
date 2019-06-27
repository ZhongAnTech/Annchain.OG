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

	"github.com/annchain/OG/types"
)

type ReceiptStatus uint8

const (
	ReceiptStatusSeqSuccess ReceiptStatus = iota
	ReceiptStatusTxSuccess
	ReceiptStatusOVMFailed
	ReceiptStatusCampaignSuccess
	ReceiptStatusTermChangeSuccess
	ReceiptStatusArchiveSuccess
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
