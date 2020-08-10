package core

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common/math"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"

	// TODO move out evm package
	evm "github.com/annchain/OG/vm/eth/core/vm"
	"github.com/annchain/OG/vm/ovm"
	vmtypes "github.com/annchain/OG/vm/types"

	log "github.com/sirupsen/logrus"
)

type OvmProcessor struct{}

func (op *OvmProcessor) CanProcess(txi types.Txi) bool {
	tx, ok := txi.(*types.Tx)
	if !ok {
		return false
	}

	_, okFrom := tx.From.(*ogTypes.Address20)
	_, okTo := tx.To.(*ogTypes.Address20)
	if !okFrom || !okTo {
		return false
	}
	return true
}

func (op *OvmProcessor) Process(statedb VmStateDB, txi types.Txi, height uint64) (*Receipt, error) {
	tx := txi.(*types.Tx)

	from20 := tx.From.(*ogTypes.Address20)
	to20 := tx.To.(*ogTypes.Address20)
	vmContext := ovm.NewOVMContext(&ovm.DefaultChainContext{}, DefaultCoinbase, statedb)

	txContext := &ovm.TxContext{
		From:       from20,
		Value:      tx.Value,
		Data:       tx.Data,
		GasPrice:   math.NewBigInt(0),
		GasLimit:   DefaultGasLimit,
		Coinbase:   DefaultCoinbase,
		SequenceID: height,
	}

	evmInterpreter := evm.NewEVMInterpreter(vmContext, txContext,
		&evm.InterpreterConfig{
			Debug: false,
		})
	ovmconf := &ovm.OVMConfig{
		NoRecursion: false,
	}
	ogvm := ovm.NewOVM(vmContext, []ovm.Interpreter{evmInterpreter}, ovmconf)

	var ret []byte
	//var leftOverGas uint64
	var contractAddress ogTypes.Address20
	var err error
	var receipt *Receipt
	if tx.To.Cmp(emptyAddress) == 0 {
		ret, contractAddress, _, err = ogvm.Create(vmtypes.AccountRef(*txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	} else {
		ret, _, err = ogvm.Call(vmtypes.AccountRef(*txContext.From), *to20, txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	}
	if err != nil {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusVMFailed, err.Error(), emptyAddress)
		log.WithError(err).Warn("vm processing error")
		return receipt, fmt.Errorf("vm processing error: %v", err)
	}
	if tx.To.Cmp(emptyAddress) == 0 {
		receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, contractAddress.Hex(), &contractAddress)
	} else {
		receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, fmt.Sprintf("%x", ret), emptyAddress)
	}
	return receipt, nil
}
