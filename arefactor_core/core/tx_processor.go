package core

import (
	"fmt"
	"github.com/annchain/OG/arefactor/types"
	"strconv"
)

type OgTxProcessor struct{}

func NewOgTxProcessor() *OgTxProcessor {
	return &OgTxProcessor{}
}

func (tp *OgTxProcessor) ProcessTransaction(engine LedgerEngine, tx types.Txi) (*Receipt, error) {
	curNonce := engine.GetNonce(tx.Sender())
	if tx.GetNonce() > curNonce {
		engine.SetNonce(tx.Sender(), tx.GetNonce())
	}

	if tx.GetType() == types.TxBaseTypeSequencer {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
		return receipt, nil
	}
	if tx.GetType() == types.TxBaseAction {
		actionTx := tx.(*types.ActionTx)
		receipt, err := ActionTxProcessor(engine, actionTx)
		if err != nil {
			return receipt, fmt.Errorf("process action tx error: %v", err)
		}
		return receipt, nil
	}

	if tx.GetType() != types.TxBaseTypeNormal {
		receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusUnknownTxType, "", emptyAddress)
		return receipt, nil
	}

	// transfer balance
	txnormal := tx.(*types.Tx)
	if txnormal.Value.Value.Sign() != 0 && !(txnormal.To.Cmp(emptyAddress) == 0) {
		engine.SubTokenBalance(txnormal.Sender(), txnormal.TokenId, txnormal.Value)
		engine.AddTokenBalance(txnormal.To, txnormal.TokenId, txnormal.Value)
	}

	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
	return receipt, nil

	//// return when its not contract related tx.
	//if len(txnormal.Data) == 0 {
	//	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
	//	return receipt, nil
	//}
	//
	//// return when the address type is not Address20
	//from20, okFrom := txnormal.From.(*ogTypes.Address20)
	//to20, okTo := txnormal.To.(*ogTypes.Address20)
	//if !okFrom || !okTo {
	//	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, "", emptyAddress)
	//	return receipt, nil
	//}
	//
	//// create ovm object.
	////
	//// TODO gaslimit not implemented yet.
	//var vmContext *vmtypes.Context
	//if preload {
	//	vmContext = ovm.NewOVMContext(&ovm.DefaultChainContext{}, DefaultCoinbase, dag.preloadDB)
	//} else {
	//	vmContext = ovm.NewOVMContext(&ovm.DefaultChainContext{}, DefaultCoinbase, dag.statedb)
	//}
	//
	//txContext := &ovm.TxContext{
	//	From:       from20,
	//	Value:      txnormal.Value,
	//	Data:       txnormal.Data,
	//	GasPrice:   math.NewBigInt(0),
	//	GasLimit:   DefaultGasLimit,
	//	Coinbase:   DefaultCoinbase,
	//	SequenceID: dag.latestSequencer.Height,
	//}
	//
	//evmInterpreter := evm.NewEVMInterpreter(vmContext, txContext,
	//	&evm.InterpreterConfig{
	//		Debug: false,
	//	})
	//ovmconf := &ovm.OVMConfig{
	//	NoRecursion: false,
	//}
	//ogvm := ovm.NewOVM(vmContext, []ovm.Interpreter{evmInterpreter}, ovmconf)
	//
	//var ret []byte
	////var leftOverGas uint64
	//var contractAddress ogTypes.Address20
	//var err error
	//var receipt *Receipt
	//if txnormal.To.Cmp(emptyAddress) == 0 {
	//	ret, contractAddress, _, err = ogvm.Create(vmtypes.AccountRef(*txContext.From), txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	//} else {
	//	ret, _, err = ogvm.Call(vmtypes.AccountRef(*txContext.From), *to20, txContext.Data, txContext.GasLimit, txContext.Value.Value, true)
	//}
	//if err != nil {
	//	receipt := NewReceipt(tx.GetTxHash(), ReceiptStatusOVMFailed, err.Error(), emptyAddress)
	//	log.WithError(err).Warn("vm processing error")
	//	return nil, receipt, fmt.Errorf("vm processing error: %v", err)
	//}
	//if txnormal.To.Cmp(emptyAddress) == 0 {
	//	receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, contractAddress.Hex(), &contractAddress)
	//} else {
	//	receipt = NewReceipt(tx.GetTxHash(), ReceiptStatusSuccess, fmt.Sprintf("%x", ret), emptyAddress)
	//}

}

func ActionTxProcessor(engine LedgerEngine, actionTx *types.ActionTx) (*Receipt, error) {

	switch actionTx.Action {
	case types.ActionTxActionIPO, types.ActionTxActionSPO, types.ActionTxActionDestroy:
		return processPublicOffering(engine, actionTx)
	default:
		return nil, fmt.Errorf("unkown tx action type: %x", actionTx.Action)
	}

}

func processPublicOffering(engine LedgerEngine, actionTx *types.ActionTx) (*Receipt, error) {

	switch offerProcess := actionTx.ActionData.(type) {
	case *types.InitialOffering:
		issuer := actionTx.Sender()
		name := offerProcess.TokenName
		reIssuable := offerProcess.EnableSPO
		amount := offerProcess.Value

		tokenID, err := engine.IssueToken(issuer, name, "", reIssuable, amount)
		if err != nil {
			receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil

	case *types.SecondaryOffering:
		tokenID := offerProcess.TokenId
		amount := offerProcess.Value

		err := engine.ReIssueToken(tokenID, amount)
		if err != nil {
			receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil

	case *types.DestroyOffering:
		tokenID := offerProcess.TokenId

		err := engine.DestroyToken(tokenID)
		if err != nil {
			receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusFailed, err.Error(), emptyAddress)
			return receipt, err
		}
		receipt := NewReceipt(actionTx.GetTxHash(), ReceiptStatusSuccess, strconv.Itoa(int(tokenID)), emptyAddress)
		return receipt, nil

	default:
		return nil, fmt.Errorf("unknown offerProcess: %v", actionTx.ActionData)
	}
}
