package core

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/OG/types"
	"math/big"
)

type TxStatusSet map[common.Hash]*TxStatus

func (t *TxStatusSet) Get(hash common.Hash) *TxStatus {
	return (*t)[hash]
}

// TODO
// try not to detect the tx type in this function
func (t *TxStatusSet) CreateStatus(txi types.Txi) {
	t.Set(txi.GetTxHash(), NewTxStatus(txi))
}

// TODO
// try not to detect the tx type in this function
func (t *TxStatusSet) BindChild(parent types.Txi, child types.Txi) {
	pHash := parent.GetTxHash()
	txStatus := t.Get(pHash)
	if txStatus == nil {
		txStatus = NewTxStatus(parent)
	}
	txStatus.AddChild(child)
	t.Set(pHash, txStatus)
}

func (t *TxStatusSet) Set(hash common.Hash, status *TxStatus) {
	(*t)[hash] = status
}

// rob simulates txs rob money from its parents.
func (t *TxStatusSet) rob(robber common.Hash, victim common.Hash, robRate int) (*math.BigInt, *math.BigInt) {

	robberStatus := t.Get(robber)
	if robberStatus == nil {
		return nil, nil
	}
	victimStatus := t.Get(victim)
	if victimStatus == nil {
		return nil, nil
	}

	robFromRobbed := big.NewInt(0).Mul(victimStatus.robbed.Value, robberStatus.guarantee.Value)
	robFromRobbed = robFromRobbed.Div(robFromRobbed, victimStatus.childrenGuarantees.Value)

	robFromGuarantee := big.NewInt(0).Mul(victimStatus.guarantee.Value, robberStatus.guarantee.Value)
	robFromGuarantee = robFromGuarantee.Div(robFromGuarantee, victimStatus.childrenGuarantees.Value)

	if robRate != 100 {
		robFromRobbed = robFromRobbed.Mul(robFromRobbed, big.NewInt(int64(robRate)))
		robFromGuarantee = robFromGuarantee.Mul(robFromGuarantee, big.NewInt(int64(100)))
	}
	robAmount := math.NewBigIntFromBigInt(big.NewInt(0).Add(robFromRobbed, robFromGuarantee))

	robberStatus.robbed = robberStatus.robbed.Add(robAmount)
	t.Set(robber, robberStatus)

	return math.NewBigIntFromBigInt(robFromRobbed),
		math.NewBigIntFromBigInt(robFromGuarantee)
}

func (t *TxStatusSet) airdrop(txHash common.Hash, seqHash common.Hash) (*math.BigInt, *math.BigInt) {
	return t.rob(txHash, seqHash, 100)
}

// forfeit simulates seq taking all of the robbed and guarantee money from its parent.
func (t *TxStatusSet) forfeit(seqHash common.Hash, txHash common.Hash, robRate int) (*math.BigInt, *math.BigInt) {
	return t.rob(seqHash, txHash, robRate)
}

type TxStatus struct {
	robbed    *math.BigInt
	guarantee *math.BigInt

	tx                 types.Txi
	children           []types.Txi
	childrenGuarantees *math.BigInt
}

func NewTxStatus(txi types.Txi) *TxStatus {
	return &TxStatus{
		robbed:             math.NewBigInt(0),
		guarantee:          txi.GetGuarantee(),
		tx:                 txi,
		children:           make([]types.Txi, 0),
		childrenGuarantees: math.NewBigInt(0),
	}
}

func (t *TxStatus) Children() []types.Txi {
	return t.children
}

func (t *TxStatus) AddChild(child types.Txi) {
	t.children = append(t.children, child)

	if child.GetGuarantee() != nil {
		t.childrenGuarantees = t.childrenGuarantees.Add(child.GetGuarantee())
	}
}
