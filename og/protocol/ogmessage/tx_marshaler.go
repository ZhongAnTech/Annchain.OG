//// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
////
//// Licensed under the Apache License, Version 2.0 (the "License");
//// you may not use this file except in compliance with the License.
//// You may obtain a copy of the License at
////
////     http://www.apache.org/licenses/LICENSE-2.0
////
//// Unless required by applicable law or agreed to in writing, software
//// distributed under the License is distributed on an "AS IS" BASIS,
//// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//// See the License for the specific language governing permissions and
//// limitations under the License.
package ogmessage

//
////do not use go gen msgp for this file  , this is written by hand
//import (
//	"encoding/binary"
//	"fmt"
//	"github.com/tinylib/msgp/msgp"
//)
//
////TxMarshaler just for marshaller , put type to front 2 bytes, and marshal
//
////TxMarshaler just for marshaller , put type to front 2 bytes, and marshal
//type RawTxMarshaler struct {
//	RawTxi `msg:"-"`
//}
//
//func (t *RawTxMarshaler) MarshalMsg(b []byte) (o []byte, err error) {
//	if t == nil || t.RawTxi == nil {
//		panic("nil txi")
//	}
//	head := make([]byte, 2)
//	binary.BigEndian.PutUint16(head, uint16(GetType()))
//	b = append(b, head...)
//	if GetType() == TxBaseAction {
//		r := t.RawTxi.(*RawActionTx)
//		b = append(b, r.Action)
//	}
//	return MarshalMsg(b)
//}
//
//func (t *RawTxMarshaler) UnmarshalMsg(bts []byte) (o []byte, err error) {
//	if len(bts) <= 3 {
//		return bts, fmt.Errorf("size mismatch")
//	}
//	tp := binary.BigEndian.Uint16(bts)
//	switch TxBaseType(tp) {
//	case TxBaseTypeNormal:
//		t.RawTxi = &RawTx{TxBase: TxBase{Type: TxBaseTypeNormal}}
//	//case TxBaseTypeCampaign:
//	//	t.RawTxi = &RawCampaign{TxBase: TxBase{Type: TxBaseTypeCampaign}}
//	//case TxBaseTypeTermChange:
//	//	t.RawTxi = &RawTermChange{TxBase: TxBase{Type: TxBaseTypeTermChange}}
//	case TxBaseTypeSequencer:
//		t.RawTxi = &RawSequencer{TxBase: TxBase{Type: TxBaseTypeSequencer}}
//	//case types.TxBaseTypeArchive:
//	//	t.RawTxi = &protocol_message.RawArchive{Archive: archive.Archive{TxBase: types.TxBase{Type: types.TxBaseTypeArchive}}}
//	case TxBaseAction:
//		rawTx := &RawActionTx{TxBase: TxBase{Type: TxBaseAction}}
//		action := bts[3]
//		if action == ActionRequestDomainName {
//			rawTx.ActionData = &RequestDomain{}
//		} else if action == ActionTxActionIPO || action == ActionTxActionSPO || action == ActionTxActionDestroy {
//			rawTx.ActionData = &PublicOffering{}
//		} else {
//			return bts, fmt.Errorf("unkown action %d", action)
//		}
//		t.RawTxi = rawTx
//		return UnmarshalMsg(bts[3:])
//	default:
//		return bts, fmt.Errorf("unkown type")
//	}
//	return UnmarshalMsg(bts[2:])
//}
//
//func (t *RawTxMarshaler) Msgsize() (s int) {
//	if GetType() == TxBaseAction {
//		return 3 + Msgsize()
//	}
//	return 2 + Msgsize()
//}
//
//func (t *RawTxMarshaler) DecodeMsg(dc *msgp.Reader) (err error) {
//	head := make([]byte, 2)
//	_, err = dc.ReadFull(head)
//	if err != nil {
//		return
//	}
//	if len(head) < 2 {
//		return fmt.Errorf("size mismatch")
//	}
//	tp := binary.BigEndian.Uint16(head)
//	switch TxBaseType(tp) {
//	case TxBaseTypeNormal:
//		t.RawTxi = &RawTx{TxBase: TxBase{Type: TxBaseTypeNormal}}
//	//case TxBaseTypeCampaign:
//	//	t.RawTxi = &RawCampaign{TxBase: TxBase{Type: TxBaseTypeCampaign}}
//	//case TxBaseTypeTermChange:
//	//	t.RawTxi = &RawTermChange{TxBase: TxBase{Type: TxBaseTypeTermChange}}
//	case TxBaseTypeSequencer:
//		t.RawTxi = &RawSequencer{TxBase: TxBase{Type: TxBaseTypeSequencer}}
//	//case types.TxBaseTypeArchive:
//	//	t.RawTxi = &protocol_message.RawArchive{Archive: archive.Archive{TxBase: types.TxBase{Type: types.TxBaseTypeArchive}}}
//	case TxBaseAction:
//		rawTx := &RawActionTx{TxBase: TxBase{Type: TxBaseAction}}
//		head := make([]byte, 1)
//		_, err := dc.ReadFull(head)
//		if err != nil {
//			return err
//		}
//		if len(head) < 1 {
//			return fmt.Errorf("size mismatch")
//		}
//		action := head[0]
//		if action == ActionRequestDomainName {
//			rawTx.ActionData = &RequestDomain{}
//		} else if action == ActionTxActionIPO || action == ActionTxActionSPO || action == ActionTxActionDestroy {
//			rawTx.ActionData = &PublicOffering{}
//		} else {
//			return fmt.Errorf("unkown action %d", action)
//		}
//		t.RawTxi = rawTx
//		return DecodeMsg(dc)
//	default:
//		return fmt.Errorf("unkown type")
//	}
//	return DecodeMsg(dc)
//}
//
//func (t *RawTxMarshaler) EncodeMsg(en *msgp.Writer) (err error) {
//	if t == nil || t.RawTxi == nil {
//		panic("nil txi")
//	}
//	head := make([]byte, 2)
//	binary.BigEndian.PutUint16(head, uint16(GetType()))
//	_, err = en.Write(head)
//	if err != nil {
//		return err
//	}
//	if GetType() == TxBaseAction {
//		r := t.RawTxi.(*RawActionTx)
//		err = en.WriteByte(r.Action)
//	}
//	if err != nil {
//		return err
//	}
//	return EncodeMsg(en)
//}
//
////func (t *RawTxMarshaler) Txi() types.Txi {
////	if t == nil || t.RawTxi == nil {
////		return nil
////	}
////	switch raw := t.RawTxi.(type) {
////	case *protocol_message.RawTx:
////		return raw.Tx()
////	case *protocol_message.RawSequencer:
////		return raw.Sequencer()
////	case *protocol_message.RawCampaign:
////		return raw.Campaign()
////	case *protocol_message.RawTermChange:
////		return raw.TermChange()
////	case *protocol_message.RawArchive:
////		return &raw.Archive
////	case *protocol_message.RawActionTx:
////		return raw.ActionTx()
////	default:
////		return nil
////	}
////}
//
//func NewTxisMarshaler(t Txis) TxisMarshaler {
//	var txs TxisMarshaler
//	for _, tx := range t {
//		txs.Append(tx)
//	}
//	if len(txs) == 0 {
//		txs = nil
//	}
//	return txs
//}
