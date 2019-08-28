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
package rpc

import (
	"encoding/hex"
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/types/tx_types"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (r *RpcController) DebugCreateContract() error {
	from := common.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	to := common.BytesToAddress(nil)
	value := math.NewBigInt(0)
	nonce := uint64(1)

	contractCode := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"
	data, _ := hex.DecodeString(contractCode)

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugQueryContract() ([]byte, error) {
	from := common.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	contractAddr := crypto.CreateAddress(from, uint64(1))

	calldata := "e5aa3d58"
	callTx := &tx_types.Tx{}
	callTx.From = &from
	callTx.Value = math.NewBigInt(0)
	callTx.To = contractAddr
	callTx.Data, _ = hex.DecodeString(calldata)

	b, _, err := r.Og.Dag.ProcessTransaction(callTx, false)
	return b, err
}

func (r *RpcController) DebugSetContract(n string) error {
	from := common.HexToAddress("0x60ce04e6a1cc8887fa5dcd43f87c38be1d41827e")
	to := crypto.CreateAddress(from, uint64(1))
	value := math.NewBigInt(0)
	curnonce, err := r.Og.Dag.GetLatestNonce(from)
	if err != nil {
		return err
	}
	nonce := curnonce + 1
	setdata := fmt.Sprintf("60fe47b1%s", n)
	data, err := hex.DecodeString(setdata)
	if err != nil {
		return err
	}

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCallerCreate() error {
	from := common.HexToAddress("0xc18969f0b7e3d192d86e220c44be2f03abdaeda2")
	to := common.BytesToAddress(nil)
	value := math.NewBigInt(0)
	nonce := uint64(1)

	contractCode := "6060604052341561000f57600080fd5b732fd82147682e011063adb8534b2d1d8831f529696000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055506000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff16600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550610247806100d46000396000f300606060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063671ac9c51461005c578063ceee2e201461007f578063f1850c1b146100d4575b600080fd5b341561006757600080fd5b61007d6004808035906020019091905050610129565b005b341561008a57600080fd5b6100926101d0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156100df57600080fd5b6100e76101f5565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166360fe47b1826040518263ffffffff167c010000000000000000000000000000000000000000000000000000000002815260040180828152602001915050600060405180830381600087803b15156101b957600080fd5b6102c65a03f115156101ca57600080fd5b50505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16815600a165627a7a72305820a1651c4d2b88805e349363d97e5141614ba060f6fd871434b856d5cc689006eb0029"
	data, _ := hex.DecodeString(contractCode)

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCallerCall(setvalue string) error {
	from := common.HexToAddress("0xc18969f0b7e3d192d86e220c44be2f03abdaeda2")
	to := crypto.CreateAddress(from, uint64(1))
	value := math.NewBigInt(0)
	curnonce, err := r.Og.Dag.GetLatestNonce(from)
	if err != nil {
		return err
	}
	nonce := curnonce + 1
	datastr := fmt.Sprintf("671ac9c5%s", setvalue)
	data, err := hex.DecodeString(datastr)
	if err != nil {
		return err
	}

	return r.DebugCreateTxAndSendToBuffer(from, to, value, data, nonce)
}

func (r *RpcController) DebugCreateTxAndSendToBuffer(from, to common.Address, value *math.BigInt, data []byte, nonce uint64) error {
	pubstr := "0x0104a391a55b84e45858748324534a187c60046266b17a5c77161104c4a6c4b1511789de692b898768ff6ee731e2fe068d6234a19a1e20246c968df9c0ca797498e7"
	pub, _ := crypto.PublicKeyFromString(pubstr)

	sigstr := "6060604052341561000f57600080fd5b600a60008190555060006001819055506102078061002e6000396000f300606060405260043610610062576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680631c0f72e11461006b57806360fe47b114610094578063c605f76c146100b7578063e5aa3d5814610145575b34600181905550005b341561007657600080fd5b61007e61016e565b6040518082815260200191505060405180910390f35b341561009f57600080fd5b6100b56004808035906020019091905050610174565b005b34156100c257600080fd5b6100ca61017e565b6040518080602001828103825283818151815260200191508051906020019080838360005b8381101561010a5780820151818401526020810190506100ef565b50505050905090810190601f1680156101375780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561015057600080fd5b6101586101c1565b6040518082815260200191505060405180910390f35b60015481565b8060008190555050565b6101866101c7565b6040805190810160405280600a81526020017f68656c6c6f576f726c6400000000000000000000000000000000000000000000815250905090565b60005481565b6020604051908101604052806000815250905600a165627a7a723058208e1bdbeee227900e60082cfcc0e44d400385e8811ae77ac6d7f3b72f630f04170029"
	sigb, _ := hex.DecodeString(sigstr)
	sig := crypto.SignatureFromBytes(crypto.CryptoTypeSecp256k1, sigb)

	tx, err := r.TxCreator.NewTxWithSeal(from, to, value, nil, data, nonce, pub, sig, 0)
	if err != nil {
		return err
	}
	r.TxBuffer.ReceivedNewTxChan <- tx
	return nil
}

func (r *RpcController) Debug(c *gin.Context) {
	p := c.Request.URL.Query().Get("f")
	switch p {
	case "1":
		r.NewRequestChan <- types.TxBaseTypeNormal
	case "2":
		r.NewRequestChan <- types.TxBaseTypeSequencer
	case "cc":
		err := r.DebugCreateContract()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("new contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "qc":
		ret, err := r.DebugQueryContract()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("query contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, fmt.Sprintf("%x", ret))
			return
		}
	case "sc":
		param := c.Request.URL.Query().Get("param")
		err := r.DebugSetContract(param)
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("set contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "callerCreate":
		err := r.DebugCallerCreate()
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("create caller contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	case "callerCall":
		value := c.Request.URL.Query().Get("value")
		err := r.DebugCallerCall(value)
		if err != nil {
			Response(c, http.StatusInternalServerError, fmt.Errorf("call caller contract failed, err: %v", err), nil)
			return
		} else {
			Response(c, http.StatusOK, nil, nil)
			return
		}
	}
}
