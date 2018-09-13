package core

import (
	"fmt"
	"github.com/annchain/OG/types"
	"github.com/annchain/OG/ogdb"
)


type SeqIndex struct {
	db ogdb.Database
}


func sequencerIdKey( id uint64) []byte {
	return []byte(fmt.Sprintf("seqId:%d",id))
}

func  txSeqKey(id uint64) []byte {
	return []byte (fmt.Sprintf("txSeqId:%d",id))
}


func NewSequenceIndex(db ogdb.Database) *SeqIndex {
	return &SeqIndex{db: db}
}
// WriteGenesis writes geneis into db.
func (s *SeqIndex) WriteSequncerHash(seq *types.Sequencer) error {

	return s.db.Put(sequencerIdKey(seq.Id), seq.Hash.ToBytes())
}

// ReadLatestSequencer get latest sequencer from db.
// return nil if there is no sequencer.
func (s *SeqIndex) ReadSequncerHash(id uint64) (hash  types.Hash,err error) {
	data, _ := s.db.Get(sequencerIdKey(id))
	if len(data) == 0 {
		return hash,fmt.Errorf("not found seq for id %d",id)
	}
	return  types.BytesToHash(data),nil
}

// WriteGenesis writes latest sequencer into db.
func (s *SeqIndex) WritetxsHashs(id uint64,  hashs types.Hashs) error {
	data, err := hashs.MarshalMsg(nil)
	if err != nil {
		return err
	}
	return s.db.Put(txSeqKey(id), data)
}

// ReadTransaction get tx or sequencer from ogdb.
func (s *SeqIndex) ReadTxHashs(id uint64 )( hashs types.Hashs,err error){
	data, _ := s.db.Get(txSeqKey(id))
	if len(data) == 0 {
		return hashs,  fmt.Errorf("not found")
	}

	_,err =  hashs.UnmarshalMsg(data)
	return
}