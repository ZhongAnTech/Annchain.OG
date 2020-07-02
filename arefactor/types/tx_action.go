package types

import (
	"fmt"
	ogTypes "github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/types"
	"math/rand"
	"strings"
	"time"
)

const (
	ActionTxActionIPO uint8 = iota
	ActionTxActionDestroy
	ActionTxActionSPO
	ActionRequestDomainName
)

type ActionData interface {
	marshaller.IMarshaller
	String() string
}

//msgp:tuple RequestDomain
type RequestDomain struct {
	DomainName string
}

//msgp:tuple ActionTx
type ActionTx struct {
	TxBase
	Action     uint8
	From       ogTypes.Address
	ActionData ActionData
	confirm    time.Time
}

func (r RequestDomain) String() string {
	return r.DomainName
}

func (t *ActionTx) GetConfirm() time.Duration {
	return time.Since(t.confirm)
}

func (t *ActionTx) Setconfirm() {
	t.confirm = time.Now()
}

func (t *ActionTx) String() string {
	if t.GetSender() == nil {
		return fmt.Sprintf("%s-[nil]-%d-ATX", t.TxBase.String(), t.AccountNonce)
	}
	return fmt.Sprintf("%s-[%.10s]-%d-ATX", t.TxBase.String(), t.Sender().AddressShortString(), t.AccountNonce)
}

func SampleActionTx() *ActionTx {
	//v, _ := math.NewBigIntFromString("-1234567890123456789012345678901234567890123456789012345678901234567890", 10)
	from, _ := ogTypes.HexToAddress20("0x99")
	hash1, _ := ogTypes.HexToHash32("0xCCDD")
	hash2, _ := ogTypes.HexToHash32("0xEEFF")
	return &ActionTx{TxBase: TxBase{
		Height:       12,
		ParentsHash:  []ogTypes.Hash{hash1, hash2},
		Type:         TxBaseTypeNormal,
		AccountNonce: 234,
	},
		From: from,
		//To:    common.HexToAddress("0x88"),
		//Value: v,
	}
}

func RandomActionTx() *ActionTx {
	from := ogTypes.RandomAddress20()
	return &ActionTx{TxBase: TxBase{
		Hash:         ogTypes.RandomHash32(),
		Height:       uint64(rand.Int63n(1000)),
		ParentsHash:  []ogTypes.Hash{ ogTypes.RandomHash32(), ogTypes.RandomHash32() },
		Type:         TxBaseTypeNormal,
		AccountNonce: uint64(rand.Int63n(50000)),
		Weight:       uint64(rand.Int31n(2000)),
	},
		From: from,
		//To:     common.RandomAddress(),
		//Value: math.NewBigInt(rand.Int63()),
	}
}

//func (t *ActionTx) GetPublicOffering() *InitialOffering {
//	if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionDestroy {
//		v, ok := t.ActionData.(*InitialOffering)
//		if ok {
//			return v
//		}
//	}
//	return nil
//}
//
//func (t *ActionTx) GetDomainName() *RequestDomain {
//	if t.Action == ActionRequestDomainName {
//		v, ok := t.ActionData.(*RequestDomain)
//		if ok {
//			return v
//		}
//	}
//	return nil
//}

func (t *ActionTx) CheckActionIsValid() bool {
	switch t.Action {
	case ActionTxActionIPO:
	case ActionTxActionSPO:
	case ActionTxActionDestroy:
	case ActionRequestDomainName:
	default:
		return false
	}
	return true
}

// TODO rewrite SignatureTargets

func (t *ActionTx) SignatureTargets() []byte {
	// log.WithField("tx", t).Tracef("SignatureTargets: %s", t.Dump())

	w := types.NewBinaryWriter()

	w.Write(t.AccountNonce, t.Action)
	if !types.CanRecoverPubFromSig {
		w.Write(t.From.Bytes)
	}
	//types.PanicIfError(binary.Write(&buf, binary.BigEndian, t.To.Bytes))
	bts, _ := t.ActionData.MarshalMsg()
	w.Write(bts)

	//if t.Action == ActionTxActionIPO || t.Action == ActionTxActionSPO || t.Action == ActionTxActionDestroy {
	//	of := t.GetPublicOffering()
	//	w.Write(of.Value.GetSigBytes(), of.EnableSPO)
	//	if t.Action == ActionTxActionIPO {
	//		w.Write([]byte(of.TokenName))
	//	} else {
	//		w.Write(of.TokenId)
	//	}
	//
	//} else if t.Action == ActionRequestDomainName {
	//	r := t.GetDomainName()
	//	w.Write(r.DomainName)
	//}
	return w.Bytes()
}

func (t *ActionTx) Sender() ogTypes.Address {
	return t.From
}

func (t *ActionTx) GetSender() ogTypes.Address {
	return t.From
}

//func (t *ActionTx) GetOfferValue() *math.BigInt {
//	return t.GetPublicOffering().Value
//}

func (t *ActionTx) Compare(tx Txi) bool {
	switch tx := tx.(type) {
	case *ActionTx:
		if t.GetTxHash().Cmp(tx.GetTxHash()) == 0 {
			return true
		}
		return false
	default:
		return false
	}
}

func (t *ActionTx) GetBase() *TxBase {
	return &t.TxBase
}

func (t *ActionTx) Dump() string {
	var phashes []string
	for _, p := range t.ParentsHash {
		phashes = append(phashes, p.Hex())
	}
	return fmt.Sprintf("hash %s, pHash:[%s], from : %s  \n nonce : %d , signatute : %s, pubkey: %s ,"+
		"height: %d , mined Nonce: %v, type: %v, weight: %d, action %d, actionData %s", t.Hash.Hex(),
		strings.Join(phashes, " ,"), t.From.Hex(),
		t.AccountNonce, hexutil.Encode(t.Signature), hexutil.Encode(t.PublicKey),
		t.Height, t.MineNonce, t.Type, t.Weight, t.Action, t.ActionData)
}

//type ActionTxs []*ActionTx
//
//func (t ActionTxs) String() string {
//	var strs []string
//	for _, v := range t {
//		strs = append(strs, v.String())
//	}
//	return strings.Join(strs, ", ")
//}

func (c *ActionTx) SetSender(addr ogTypes.Address) {
	c.From = addr
}

/**
marshaller part
 */

func (c *ActionTx) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)

	// TxBase
	b, err := marshaller.AppendIMarshaller(b, &c.TxBase)
	if err != nil {
		return b, err
	}
	// uint8 Action
	b = marshaller.AppendUint8(b, c.Action)
	// Address from
	b, err = marshaller.AppendIMarshaller(b, c.From)
	if err != nil {
		return b, err
	}
	// ActionData ActionData
	b, err = marshaller.AppendIMarshaller(b, c.ActionData)
	if err != nil {
		return b, err
	}

	return b, nil
}

func (c *ActionTx) UnmarshalMsg(b []byte) ([]byte, error) {
	var err error

	b, _, err = marshaller.DecodeHeader(b)
	if err != nil {
		return nil, err
	}

	var txBase TxBase
	b, err = txBase.UnmarshalMsg(b)
	if err != nil {
		return b, err
	}
	c.TxBase = txBase

	c.Action, b, err = marshaller.ReadUint8(b)
	if err != nil {
		return b, err
	}

	c.From, b, err = ogTypes.UnmarshalAddress(b)
	if err != nil {
		return b, err
	}

	var actionData ActionData
	b, err = actionData.UnmarshalMsg(b)
	if err != nil {
		return b, err
	}
	c.ActionData = actionData

	return b, nil
}

func (c *ActionTx) MsgSize() int {
	return marshaller.Uint8Size +
		marshaller.CalIMarshallerSize(c.TxBase.MsgSize()) +
		marshaller.CalIMarshallerSize(c.From.MsgSize()) +
		marshaller.CalIMarshallerSize(c.ActionData.MsgSize())
}
