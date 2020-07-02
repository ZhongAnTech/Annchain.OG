package types

import (
	"fmt"
	"github.com/annchain/OG/arefactor/utils/marshaller"
	"github.com/annchain/OG/common/math"
)

type InitialOffering struct {
	Value   *math.BigInt `json:"value"`
	EnableSPO bool   `json:"enable_spo"` //if enableSPO is false  , no Secondary Public Issues.
	TokenName string `json:"token_name"`
}

func NewInitialOffering(value *math.BigInt, enableSPO bool, tokenName string) *InitialOffering {
	return &InitialOffering{
		Value: value,
		EnableSPO: enableSPO,
		TokenName: tokenName,
	}
}

func (p InitialOffering) String() string {
	return fmt.Sprintf("IPO tokenName %s,value %v, EnableSPO %v", p.TokenName, p.Value, p.EnableSPO)
}

func (p *InitialOffering) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)

	// BigInt Value
	b = marshaller.AppendBigInt(b, p.Value.Value)
	// bool EnableSPO
	b = marshaller.AppendBool(b, p.EnableSPO)
	// string  TokenName
	b = marshaller.AppendString(b, p.TokenName)
	// header
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (p *InitialOffering) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return b, err
	}

	// Value
	value, b, err := marshaller.ReadBigInt(b)
	if err != nil {
		return b, err
	}
	p.Value = math.NewBigIntFromBigInt(value)
	// EnableSPO
	p.EnableSPO, b, err = marshaller.ReadBool(b)
	if err != nil {
		return b, err
	}
	// TokenName
	p.TokenName, b, err = marshaller.ReadString(b)
	if err != nil {
		return b, err
	}

	return b, nil
}

func (p *InitialOffering) MsgSize() int {
	return marshaller.CalBigIntSize(p.Value.Value) + 1 + marshaller.CalStringSize(p.TokenName)
}

/**
SecondaryOffering
 */

type SecondaryOffering struct {
	TokenId int32        `json:"token_id"` //for Secondary Public Issues
	Value   *math.BigInt `json:"value"`
}

func NewSecondaryOffering(tokenId int32, value *math.BigInt) *SecondaryOffering {
	return &SecondaryOffering{
		TokenId: tokenId,
		Value: math.NewBigInt(0),
	}
}

func (so SecondaryOffering) String() string {
	return fmt.Sprintf("SPO tokenID %d,value %v", so.TokenId, so.Value)
}

func (so *SecondaryOffering) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)

	// int32 TokenId
	b = marshaller.AppendInt32(b, so.TokenId)
	// BigInt Value
	b = marshaller.AppendBigInt(b, so.Value.Value)
	// header
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (so *SecondaryOffering) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return b, err
	}

	// TokenId
	so.TokenId, b, err = marshaller.ReadInt32(b)
	if err != nil {
		return b, err
	}
	// Value
	value, b, err := marshaller.ReadBigInt(b)
	if err != nil {
		return b, err
	}
	so.Value = math.NewBigIntFromBigInt(value)

	return b, nil
}

func (so *SecondaryOffering) MsgSize() int {
	return marshaller.Int32Size + marshaller.CalBigIntSize(so.Value.Value)
}

/**
DestroyOffering
 */

type DestroyOffering struct {
	TokenId int32        `json:"token_id"`
}

func NewDestroyOffering(tokenId int32) *DestroyOffering {
	return &DestroyOffering{
		TokenId: tokenId,
	}
}

func (d *DestroyOffering) String() string {
	return fmt.Sprintf("Destroy tokenID %d", d.TokenId)
}

func (d *DestroyOffering) MarshalMsg() ([]byte, error) {
	b := make([]byte, marshaller.HeaderSize)

	// int32 TokenId
	b = marshaller.AppendInt32(b, d.TokenId)
	// header
	b = marshaller.FillHeaderData(b)

	return b, nil
}

func (d *DestroyOffering) UnmarshalMsg(b []byte) ([]byte, error) {
	b, _, err := marshaller.DecodeHeader(b)
	if err != nil {
		return b, err
	}

	// TokenId
	d.TokenId, b, err = marshaller.ReadInt32(b)
	if err != nil {
		return b, err
	}

	return b, nil
}

func (d *DestroyOffering) MsgSize() int {
	return marshaller.Int32Size
}


