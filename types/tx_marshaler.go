package types

//do not use go gen msgp for this file  , this is written by hand
import (
	"encoding/binary"
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

//TxMarshaler just for marshaller , put type to front 2 bytes, and marshal

type RawTxi interface {
	GetType() TxBaseType
	GetHeight() uint64
	GetWeight() uint64
	GetTxHash() Hash
	GetNonce() uint64
	Parents() Hashes // Parents returns the hash of txs that it directly proves.
	SetHash(h Hash)
	String() string
	CalcTxHash() Hash    // TxHash returns a full tx hash (parents sealed by PoW stage 2)
	CalcMinedHash() Hash // NonceHash returns the part that needs to be considered in PoW stage 1.
	CalculateWeight(parents Txis) uint64

	Txi() Txi

	// implemented by msgp
	DecodeMsg(dc *msgp.Reader) (err error)
	EncodeMsg(en *msgp.Writer) (err error)
	MarshalMsg(b []byte) (o []byte, err error)
	UnmarshalMsg(bts []byte) (o []byte, err error)
	Msgsize() (s int)
}

//TxMarshaler just for marshaller , put type to front 2 bytes, and marshal
type RawTxMarshaler struct {
	RawTxi `msg:"-"`
}

func (t *RawTxMarshaler) MarshalMsg(b []byte) (o []byte, err error) {
	if t == nil || t.RawTxi == nil {
		panic("nil txi")
	}
	head := make([]byte, 2)
	binary.BigEndian.PutUint16(head, uint16(t.GetType()))
	b = append(b, head...)
	return t.RawTxi.MarshalMsg(b)
}

func (t *RawTxMarshaler) UnmarshalMsg(bts []byte) (o []byte, err error) {
	if len(bts) < 3 {
		return bts, fmt.Errorf("size mismatch")
	}
	tp := binary.BigEndian.Uint16(bts)
	switch TxBaseType(tp) {
	case TxBaseTypeNormal:
		t.RawTxi = &RawTx{TxBase: TxBase{Type: TxBaseTypeNormal}}
	case TxBaseTypeCampaign:
		t.RawTxi = &RawCampaign{TxBase: TxBase{Type: TxBaseTypeCampaign}}
	case TxBaseTypeTermChange:
		t.RawTxi = &RawTermChange{TxBase: TxBase{Type: TxBaseTypeTermChange}}
	case TxBaseTypeSequencer:
		t.RawTxi = &RawSequencer{TxBase: TxBase{Type: TxBaseTypeSequencer}}
	default:
		return bts, fmt.Errorf("unkown type")
	}
	return t.RawTxi.UnmarshalMsg(bts[2:])
}

func (t *RawTxMarshaler) Msgsize() (s int) {
	return 2 + t.RawTxi.Msgsize()
}

func (t *RawTxMarshaler) DecodeMsg(dc *msgp.Reader) (err error) {
	head := make([]byte, 2)
	_, err = dc.ReadFull(head)
	if err != nil {
		return
	}
	if len(head) < 2 {
		return fmt.Errorf("size mismatch")
	}
	tp := binary.BigEndian.Uint16(head)
	switch TxBaseType(tp) {
	case TxBaseTypeNormal:
		t.RawTxi = &RawTx{TxBase: TxBase{Type: TxBaseTypeNormal}}
	case TxBaseTypeCampaign:
		t.RawTxi = &RawCampaign{TxBase: TxBase{Type: TxBaseTypeCampaign}}
	case TxBaseTypeTermChange:
		t.RawTxi = &RawTermChange{TxBase: TxBase{Type: TxBaseTypeTermChange}}
	case TxBaseTypeSequencer:
		t.RawTxi = &RawSequencer{TxBase: TxBase{Type: TxBaseTypeSequencer}}
	default:
		return fmt.Errorf("unkown type")
	}
	return t.RawTxi.DecodeMsg(dc)
}

func (t *RawTxMarshaler) EncodeMsg(en *msgp.Writer) (err error) {
	if t == nil || t.RawTxi == nil {
		panic("nil txi")
	}
	head := make([]byte, 2)
	binary.BigEndian.PutUint16(head, uint16(t.GetType()))
	_, err = en.Write(head)
	if err != nil {
		return err
	}
	return t.RawTxi.EncodeMsg(en)
}

func (t *RawTxMarshaler) Txi() Txi {
	if t == nil || t.RawTxi == nil {
		return nil
	}
	switch raw := t.RawTxi.(type) {
	case *RawTx:
		return raw.Tx()
	case *RawSequencer:
		return raw.Sequencer()
	case *RawCampaign:
		return raw.Campaign()
	case *RawTermChange:
		return raw.TermChange()
	default:
		return nil
	}
}
