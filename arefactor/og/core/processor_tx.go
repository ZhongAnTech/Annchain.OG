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

func (tp *OgTxProcessor) Process(engine LedgerEngine, tx types.Txi) (*Receipt, error) {
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
