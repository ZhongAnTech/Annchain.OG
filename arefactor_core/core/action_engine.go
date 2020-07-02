package core

import (
	"fmt"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/types"
	"github.com/annchain/OG/common/math"
	"strconv"
)

type ActionEngine interface {
	IssueToken(issuer ogTypes.Address, name, symbol string, reIssuable bool, fstIssue *math.BigInt) (int32, error)
	ReIssueToken(tokenID int32, amount *math.BigInt) error
	DestroyToken(tokenID int32) error
}

func ActionTxProcessor(engine ActionEngine, actionTx *types.ActionTx) (*Receipt, error) {

	switch actionTx.Action {
	case types.ActionTxActionIPO, types.ActionTxActionSPO, types.ActionTxActionDestroy:
		return processPublicOffering(engine, actionTx)
	default:
		return nil, fmt.Errorf("unkown tx action type: %x", actionTx.Action)
	}

}

func processPublicOffering(engine ActionEngine, actionTx *types.ActionTx) (*Receipt, error) {

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