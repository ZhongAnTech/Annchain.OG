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
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/utils/marshaller"
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
	TxHash          ogTypes.Hash
	Status          ReceiptStatus
	ProcessResult   string
	ContractAddress ogTypes.Address
}

func NewReceipt(hash ogTypes.Hash, status ReceiptStatus, pResult string, addr ogTypes.Address) *Receipt {
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

/**
marshalling part
 */

func (r *Receipt) MarshalMsg() ([]byte, error) {
	var err error
	b := make([]byte, marshaller.HeaderSize)

	// Hash TxHash
	b, err = marshaller.AppendIMarshaller(b, r.TxHash)
	if err != nil {
		return nil, err
	}
	// uint8 Status
	b = append(b, byte(r.Status))
	// string ProcessResult
	b = marshaller.AppendString(b, r.ProcessResult)
	// Address ContractAddress
	b, err = marshaller.AppendIMarshaller(b, r.ContractAddress)
	if err != nil {
		return nil, err
	}

	b = marshaller.FillHeaderData(b)
	return b, nil
}

func (r *Receipt) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}

	r.TxHash, b, err = ogTypes.UnmarshalHash(b)
	if err != nil {
		return nil, err
	}

	r.Status = ReceiptStatus(b[0])
	b = b[1:]

	r.ProcessResult, b, err = marshaller.ReadString(b)
	if err != nil {
		return nil, err
	}

	r.ContractAddress, b, err = ogTypes.UnmarshalAddress(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (r *Receipt) MsgSize() int {
	size := 0

	size += marshaller.CalIMarshallerSize(r.TxHash.MsgSize()) + 1 +		// TxHash + Status
		marshaller.CalStringSize(r.ProcessResult) +						// ProcessResult
		marshaller.CalIMarshallerSize(r.ContractAddress.MsgSize())		// ContractAddress

	return size
}

//msgp:tuple ReceiptSet
type ReceiptSet map[string]*Receipt
