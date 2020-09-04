package ogsyncer_interface

import (
	"fmt"
	"github.com/annchain/commongo/hexutil"
	"strconv"
)

//go:generate msgp

type OgSyncMessageType int

const (
	OgSyncMessageTypeLatestHeightRequest OgSyncMessageType = iota + 20
	OgSyncMessageTypeLatestHeightResponse
	OgSyncMessageTypeByHashesRequest
	OgSyncMessageTypeBlockByHeightRequest
	OgSyncMessageTypeBlockByHashRequest

	OgSyncMessageTypeByHashesResponse
	OgSyncMessageTypeBlockByHeightResponse
	OgSyncMessageTypeByBlockHashResponse

	OgAnnouncementTypeNewSequencer
	OgAnnouncementTypeNewTx
	OgAnnouncementTypeNewTxs
	OgAnnouncementTypeNewInt
)

var MapMessageType = map[OgSyncMessageType]string{
	OgSyncMessageTypeLatestHeightRequest:   "OgSyncMessageTypeLatestHeightRequest",
	OgSyncMessageTypeLatestHeightResponse:  "OgSyncMessageTypeLatestHeightResponse",
	OgSyncMessageTypeByHashesRequest:       "OgSyncMessageTypeByHashesRequest",
	OgSyncMessageTypeBlockByHeightRequest:  "OgSyncMessageTypeBlockByHeightRequest",
	OgSyncMessageTypeBlockByHashRequest:    "OgSyncMessageTypeBlockByHashRequest",
	OgSyncMessageTypeByHashesResponse:      "OgSyncMessageTypeByHashesResponse",
	OgSyncMessageTypeBlockByHeightResponse: "OgSyncMessageTypeBlockByHeightResponse",
	OgSyncMessageTypeByBlockHashResponse:   "OgSyncMessageTypeByBlockHashResponse",
	OgAnnouncementTypeNewSequencer:         "OgAnnouncementTypeNewSequencer",
	OgAnnouncementTypeNewTx:                "OgAnnouncementTypeNewTx",
	OgAnnouncementTypeNewTxs:               "OgAnnouncementTypeNewTxs",
	OgAnnouncementTypeNewInt:               "OgAnnouncementTypeNewInt",
}

func (o OgSyncMessageType) String() string {
	if v, ok := MapMessageType[o]; ok {
		return v
	} else {
		return "Unknown Message " + strconv.Itoa(int(o))

	}
}

//msgp OgSyncLatestHeightRequest
type OgSyncLatestHeightRequest struct {
}

func (z *OgSyncLatestHeightRequest) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncLatestHeightRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncLatestHeightRequest) GetType() OgSyncMessageType {
	return OgSyncMessageTypeLatestHeightRequest
}

func (z *OgSyncLatestHeightRequest) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncLatestHeightRequest) String() string {
	return fmt.Sprintf("OgSyncLatestHeightRequest")
}

//msgp OgSyncLatestHeightResponse
type OgSyncLatestHeightResponse struct {
	MyHeight int64
}

func (z *OgSyncLatestHeightResponse) GetType() OgSyncMessageType {
	return OgSyncMessageTypeLatestHeightResponse
}

func (z *OgSyncLatestHeightResponse) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncLatestHeightResponse) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncLatestHeightResponse) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncLatestHeightResponse) String() string {
	return fmt.Sprintf("OgSyncLatestHeightResponse [%d]", z.MyHeight)
}

//msgp OgSyncByHashesRequest
type OgSyncByHashesRequest struct {
	Hashes [][]byte
}

func (z *OgSyncByHashesRequest) GetType() OgSyncMessageType {
	return OgSyncMessageTypeByHashesRequest
}

func (z *OgSyncByHashesRequest) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncByHashesRequest) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncByHashesRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncByHashesRequest) String() string {
	return fmt.Sprintf("OgSyncByHashesRequest len [%d]", len(z.Hashes))
}

//msgp OgSyncByHashesResponse
type OgSyncByHashesResponse struct {
	Sequencers []MessageContentSequencer
	Txs        []MessageContentTx
	Ints       []MessageContentInt
}

func (z *OgSyncByHashesResponse) GetType() OgSyncMessageType {
	return OgSyncMessageTypeByHashesResponse
}

func (z *OgSyncByHashesResponse) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncByHashesResponse) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncByHashesResponse) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncByHashesResponse) String() string {
	return fmt.Sprintf("OgSyncByHashesResponse seq [%d] txs [%d] Int [%d]",
		len(z.Sequencers), len(z.Txs), len(z.Ints))
}

//msgp OgSyncBlockByHeightRequest
type OgSyncBlockByHeightRequest struct {
	Height int64
	Offset int
}

func (z *OgSyncBlockByHeightRequest) GetType() OgSyncMessageType {
	return OgSyncMessageTypeBlockByHeightRequest
}

func (z *OgSyncBlockByHeightRequest) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncBlockByHeightRequest) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncBlockByHeightRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncBlockByHeightRequest) String() string {
	return fmt.Sprintf("OgSyncBlockByHeightRequest height [%d] offset [%d]", z.Height, z.Offset)

}

//msgp OgSyncBlockByHeightResponse
type OgSyncBlockByHeightResponse struct {
	HasMore    bool
	Sequencers []MessageContentSequencer
	Ints       []MessageContentInt
	Txs        []MessageContentTx
}

func (z *OgSyncBlockByHeightResponse) GetType() OgSyncMessageType {
	return OgSyncMessageTypeBlockByHeightResponse
}

func (z *OgSyncBlockByHeightResponse) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncBlockByHeightResponse) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncBlockByHeightResponse) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncBlockByHeightResponse) String() string {
	return fmt.Sprintf("OgSyncByHashesResponse seq [%d] txs [%d] Ints [%d]",
		len(z.Sequencers), len(z.Txs), len(z.Ints))
}

//msgp OgSyncBlockByHashRequest
type OgSyncBlockByHashRequest struct {
	Hash   []byte
	Offset int
}

func (z *OgSyncBlockByHashRequest) GetType() OgSyncMessageType {
	return OgSyncMessageTypeBlockByHashRequest
}

func (z *OgSyncBlockByHashRequest) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncBlockByHashRequest) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncBlockByHashRequest) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncBlockByHashRequest) String() string {
	return fmt.Sprintf("OgSyncLatestHeightResponse hashes [%d] offset [%d]", len(z.Hash), z.Offset)
}

//msgp OgSyncBlockByHashResponse
type OgSyncBlockByHashResponse struct {
	HasMore    bool
	Sequencers []MessageContentSequencer
	Ints       []MessageContentInt
	Txs        []MessageContentTx
}

func (z *OgSyncBlockByHashResponse) GetType() OgSyncMessageType {
	return OgSyncMessageTypeByBlockHashResponse
}

func (z *OgSyncBlockByHashResponse) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgSyncBlockByHashResponse) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgSyncBlockByHashResponse) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgSyncBlockByHashResponse) String() string {
	return fmt.Sprintf("OgSyncBlockByHashResponse seq [%d] txs [%d] Int [%d]",
		len(z.Sequencers), len(z.Txs), len(z.Ints))
}

//msgp OgAnnouncementNewSequencer
type OgAnnouncementNewSequencer struct {
	Sequencer MessageContentSequencer
}

func (z *OgAnnouncementNewSequencer) GetType() OgSyncMessageType {
	return OgAnnouncementTypeNewSequencer
}

func (z *OgAnnouncementNewSequencer) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgAnnouncementNewSequencer) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgAnnouncementNewSequencer) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgAnnouncementNewSequencer) String() string {
	return fmt.Sprintf("Seq_%d", z.Sequencer.Height)
}

//msgp OgAnnouncementNewTx
type OgAnnouncementNewTx struct {
	Tx MessageContentTx
}

func (z *OgAnnouncementNewTx) GetType() OgSyncMessageType {
	return OgAnnouncementTypeNewTx
}

func (z *OgAnnouncementNewTx) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgAnnouncementNewTx) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgAnnouncementNewTx) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgAnnouncementNewTx) String() string {
	return fmt.Sprintf("Tx_%s", hexutil.Encode(z.Tx.Hash))
}

//msgp OgAnnouncementNewInt
type OgAnnouncementNewInt struct {
	Ints MessageContentInt
}

func (z *OgAnnouncementNewInt) GetType() OgSyncMessageType {
	return OgAnnouncementTypeNewInt
}

func (z *OgAnnouncementNewInt) GetTypeValue() int {
	return int(z.GetType())
}

func (z *OgAnnouncementNewInt) ToBytes() []byte {
	b, err := z.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	return b
}

func (z *OgAnnouncementNewInt) FromBytes(bts []byte) error {
	_, err := z.UnmarshalMsg(bts)
	if err != nil {
		return err
	}
	return nil
}

func (z *OgAnnouncementNewInt) String() string {
	return fmt.Sprintf("Ints_%d", z.Ints.Height)
}
