package types

import (
	"fmt"
)

//go:generate msgp

type Message interface {
	MarshalMsg([]byte) ([]byte, error)
	String()string      //string is for logging ,
	UnmarshalMsg([]byte) ([]byte, error)
}

//msgp:tuple MessageSyncRequest
type MessageSyncRequest struct {
	Hashes []Hash
}

func (m *MessageSyncRequest) String() string {
	return HashesToString(m.Hashes)
}

//msgp:tuple MessageSyncResponse
type MessageSyncResponse struct {
	Txs        []*Tx
	Sequencers []*Sequencer
}

func (m *MessageSyncResponse) String() string {
	return fmt.Sprintf("txs: [%s], seqs: [%s]", TxsToString(m.Txs), SeqsToString(m.Sequencers))
}

//msgp:tuple MessageNewTx
type MessageNewTx struct {
	Tx *Tx
}

func (m *MessageNewTx) String() string {
	return m.Tx.String()
}

//msgp:tuple MessageNewSequencer
type MessageNewSequencer struct {
	Sequencer *Sequencer
}

func (m *MessageNewSequencer) String() string {
	return m.Sequencer.String()
}

//msgp:tuple MessageNewTxs
type MessageNewTxs struct {
	Txs []*Tx
}

func (m *MessageNewTxs) String() string {
	return TxsToString(m.Txs)
}

//msgp:tuple MessageTxsRequest
type MessageTxsRequest struct {
	Hashes  []Hash
	SeqHash *Hash
	Id      uint64
}


func ( m*MessageTxsRequest) String () string{
	return fmt.Sprintf("hashes: [%s], seqHash: %s, id : %d,", HashesToString(m.Hashes),m.SeqHash.String(),m.Id)
}
//msgp:tuple MessageTxsResponse
type MessageTxsResponse struct {
	Txs       []*Tx
	Sequencer *Sequencer
}

func ( m*MessageTxsResponse) String () string{
	return fmt.Sprintf("txs: [%s], Sequencer: %s", TxsToString(m.Txs),m.Sequencer.String())
}

// getBlockHeadersData represents a block header query.
//msgp:tuple MessageHeaderRequest
type MessageHeaderRequest struct {
	Origin  HashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

func ( m*MessageHeaderRequest) String () string{
	return fmt.Sprintf("Origin: [%s],amount : %d ,skip : %d, reverse : %v, ",m.Origin.String() ,m.Amount,m.Skip,m.Reverse)
}

// hashOrNumber is a combined field for specifying an origin block.
//msgp:tuple HashOrNumber
type HashOrNumber struct {
	Hash   Hash   // Block hash from which to retrieve headers (excludes Number)
	Number uint64 // Block hash from which to retrieve headers (excludes Hash)
}

func ( m*HashOrNumber) String () string{
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(),m.Number)
}

//msgp:tuple MessageSequencerHeader
type MessageSequencerHeader struct {
	Hash   *Hash
	Number uint64
}

func ( m*MessageSequencerHeader) String () string{
	return fmt.Sprintf("hash: %s, number : %d", m.Hash.String(),m.Number)
}

//msgp:tuple MessageHeaderResponse
type MessageHeaderResponse struct {
	Sequencers []*Sequencer
}

func ( m*MessageHeaderResponse) String () string{
	return fmt.Sprintf("seqs: [%s]",SeqsToString(m.Sequencers))
}

//msgp:tuple MessageBodiesRequest
type MessageBodiesRequest struct {
	SeqHashes []Hash
}

func ( m*MessageBodiesRequest) String () string{
	return HashesToString(m.SeqHashes)
}

//msgp:tuple MessageBodiesResponse
type MessageBodiesResponse struct {
	Bodies []RawData
}

func ( m*MessageBodiesResponse) String () string{
	return fmt.Sprintf("bodies len : %d",len(m.Bodies))
}

type RawData []byte
