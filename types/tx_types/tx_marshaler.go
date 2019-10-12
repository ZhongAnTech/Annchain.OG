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
package tx_types

//do not use go gen msgp for this file  , this is written by hand
import (
	"encoding/binary"
	"fmt"
	"github.com/annchain/OG/og/protocol_message"
	"github.com/annchain/OG/types"
	"github.com/tinylib/msgp/msgp"
)

//TxMarshaler just for marshaller , put type to front 2 bytes, and marshal

//TxMarshaler just for marshaller , put type to front 2 bytes, and marshal
type RawTxMarshaler struct {
	types.RawTxi `msg:"-"`
}

func (t *RawTxMarshaler) MarshalMsg(b []byte) (o []byte, err error) {
	if t == nil || t.RawTxi == nil {
		panic("nil txi")
	}
	head := make([]byte, 2)
	binary.BigEndian.PutUint16(head, uint16(t.GetType()))
	b = append(b, head...)
	if t.GetType() == types.TxBaseAction {
		r := t.RawTxi.(*protocol_message.RawActionTx)
		b = append(b, r.Action)
	}
	return t.RawTxi.MarshalMsg(b)
}

func (t *RawTxMarshaler) UnmarshalMsg(bts []byte) (o []byte, err error) {
	if len(bts) <= 3 {
		return bts, fmt.Errorf("size mismatch")
	}
	tp := binary.BigEndian.Uint16(bts)
	switch types.TxBaseType(tp) {
	case types.TxBaseTypeNormal:
		t.RawTxi = &protocol_message.RawTx{TxBase: types.TxBase{Type: types.TxBaseTypeNormal}}
	case types.TxBaseTypeCampaign:
		t.RawTxi = &protocol_message.RawCampaign{TxBase: types.TxBase{Type: types.TxBaseTypeCampaign}}
	case types.TxBaseTypeTermChange:
		t.RawTxi = &protocol_message.RawTermChange{TxBase: types.TxBase{Type: types.TxBaseTypeTermChange}}
	case types.TxBaseTypeSequencer:
		t.RawTxi = &protocol_message.RawSequencer{TxBase: types.TxBase{Type: types.TxBaseTypeSequencer}}
	case types.TxBaseTypeArchive:
		t.RawTxi = &protocol_message.RawArchive{Archive: Archive{TxBase: types.TxBase{Type: types.TxBaseTypeArchive}}}
	case types.TxBaseAction:
		rawTx := &protocol_message.RawActionTx{TxBase: types.TxBase{Type: types.TxBaseAction}}
		action := bts[3]
		if action == ActionRequestDomainName {
			rawTx.ActionData = &RequestDomain{}
		} else if action == ActionTxActionIPO || action == ActionTxActionSPO || action == ActionTxActionDestroy {
			rawTx.ActionData = &PublicOffering{}
		} else {
			return bts, fmt.Errorf("unkown action %d", action)
		}
		t.RawTxi = rawTx
		return t.RawTxi.UnmarshalMsg(bts[3:])
	default:
		return bts, fmt.Errorf("unkown type")
	}
	return t.RawTxi.UnmarshalMsg(bts[2:])
}

func (t *RawTxMarshaler) Msgsize() (s int) {
	if t.GetType() == types.TxBaseAction {
		return 3 + t.RawTxi.Msgsize()
	}
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
	switch types.TxBaseType(tp) {
	case types.TxBaseTypeNormal:
		t.RawTxi = &protocol_message.RawTx{TxBase: types.TxBase{Type: types.TxBaseTypeNormal}}
	case types.TxBaseTypeCampaign:
		t.RawTxi = &protocol_message.RawCampaign{TxBase: types.TxBase{Type: types.TxBaseTypeCampaign}}
	case types.TxBaseTypeTermChange:
		t.RawTxi = &protocol_message.RawTermChange{TxBase: types.TxBase{Type: types.TxBaseTypeTermChange}}
	case types.TxBaseTypeSequencer:
		t.RawTxi = &protocol_message.RawSequencer{TxBase: types.TxBase{Type: types.TxBaseTypeSequencer}}
	case types.TxBaseTypeArchive:
		t.RawTxi = &protocol_message.RawArchive{Archive: Archive{TxBase: types.TxBase{Type: types.TxBaseTypeArchive}}}
	case types.TxBaseAction:
		rawTx := &protocol_message.RawActionTx{TxBase: types.TxBase{Type: types.TxBaseAction}}
		head := make([]byte, 1)
		_, err := dc.ReadFull(head)
		if err != nil {
			return err
		}
		if len(head) < 1 {
			return fmt.Errorf("size mismatch")
		}
		action := head[0]
		if action == ActionRequestDomainName {
			rawTx.ActionData = &RequestDomain{}
		} else if action == ActionTxActionIPO || action == ActionTxActionSPO || action == ActionTxActionDestroy {
			rawTx.ActionData = &PublicOffering{}
		} else {
			return fmt.Errorf("unkown action %d", action)
		}
		t.RawTxi = rawTx
		return t.RawTxi.DecodeMsg(dc)
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
	if t.GetType() == types.TxBaseAction {
		r := t.RawTxi.(*protocol_message.RawActionTx)
		err = en.WriteByte(r.Action)
	}
	if err != nil {
		return err
	}
	return t.RawTxi.EncodeMsg(en)
}

func (t *RawTxMarshaler) Txi() types.Txi {
	if t == nil || t.RawTxi == nil {
		return nil
	}
	switch raw := t.RawTxi.(type) {
	case *protocol_message.RawTx:
		return raw.Tx()
	case *protocol_message.RawSequencer:
		return raw.Sequencer()
	case *protocol_message.RawCampaign:
		return raw.Campaign()
	case *protocol_message.RawTermChange:
		return raw.TermChange()
	case *protocol_message.RawArchive:
		return &raw.Archive
	case *protocol_message.RawActionTx:
		return raw.ActionTx()
	default:
		return nil
	}
}

func NewTxisMarshaler(t types.Txis) protocol_message.TxisMarshaler {
	var txs protocol_message.TxisMarshaler
	for _, tx := range t {
		txs.Append(tx)
	}
	if len(txs) == 0 {
		txs = nil
	}
	return txs
}
