package ledger

import (
	"errors"
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/og/types"
)

// TxiLedgerMarshaller let ledger know how to store and low Txi
type TxiLedgerMarshaller struct {
}

func (t TxiLedgerMarshaller) ToBytes(txi types.Txi) []byte {
	switch txi.GetType() {
	case types.TxBaseTypeTx:
		tx := txi.(*types.Tx)
		ledgerObject := LedgerContentTx{
			Hash:         tx.Hash,
			ParentsHash:  tx.ParentsHash,
			MineNonce:    tx.MineNonce,
			AccountNonce: tx.AccountNonce,
			From:         tx.From,
			To:           tx.To,
			Value:        tx.Value,
			TokenId:      tx.TokenId,
			PublicKey:    tx.PublicKey.ToBytes(),
			Data:         tx.Data,
			Signature:    tx.Signature.ToBytes(),
		}
		return ledgerObject.ToBytes()
	case types.TxBaseTypeSequencer:
		seq := txi.(*types.Sequencer)
		ledgerObject := LedgerContentSequencer{
			Hash:         seq.Hash,
			ParentsHash:  seq.ParentsHash,
			MineNonce:    seq.MineNonce,
			AccountNonce: seq.AccountNonce,
			Issuer:       seq.Issuer,
			PublicKey:    seq.PublicKey,
			Signature:    seq.Signature,
			StateRoot:    seq.StateRoot,
			Height:       seq.Height,
		}
		return ledgerObject.ToBytes()
	default:
		panic("unknown tx type in txi_binary:ToBytes")
	}
}

func (t TxiLedgerMarshaller) FromBytes(bts []byte, bType types.TxBaseType) (tx types.Txi, err error) {
	switch bType {
	case types.TxBaseTypeTx:
		txdb := &LedgerContentTx{}
		err = txdb.FromBytes(bts)
		if err != nil {
			return
		}
		tx = &types.Tx{
			Hash:         txdb.Hash,
			ParentsHash:  txdb.ParentsHash,
			MineNonce:    txdb.MineNonce,
			AccountNonce: txdb.AccountNonce,
			From:         txdb.From,
			To:           txdb.To,
			Value:        txdb.Value,
			TokenId:      txdb.TokenId,
			Data:         txdb.Data,
			PublicKey:    crypto.PublicKeyFromRawBytes(txdb.PublicKey),
			Signature:    crypto.SignatureFromRawBytes(txdb.Signature),
			Height:       txdb.Height,
			Weight:       txdb.Weight,
		}
		return
	case types.TxBaseTypeSequencer:
		txdb := &LedgerContentSequencer{}
		err = txdb.FromBytes(bts)
		if err != nil {
			return
		}
		tx = &types.Sequencer{
			Hash:         txdb.Hash,
			ParentsHash:  txdb.ParentsHash,
			Height:       txdb.Height,
			MineNonce:    txdb.MineNonce,
			AccountNonce: txdb.AccountNonce,
			Issuer:       txdb.Issuer,
			Signature:    txdb.Signature,
			PublicKey:    txdb.PublicKey,
			StateRoot:    txdb.StateRoot,
			Weight:       txdb.Weight,
		}
		return
	default:
		err = errors.New("unknown tx type in txi_binary:FromBytes")
		return
	}
}