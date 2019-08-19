package message

import "github.com/tinylib/msgp/msgp"

// DecodeMsg implements msgp.Decodable
func (z *MessageConsensusUnmarshaller) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z MessageConsensusUnmarshaller) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 0
	err = en.Append(0x80)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MessageConsensusUnmarshaller) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 0
	o = append(o, 0x80)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MessageConsensusUnmarshaller) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z MessageConsensusUnmarshaller) Msgsize() (s int) {
	s = 1
	return
}


// DecodeMsg implements msgp.Decodable
func (z *SequencerProposal) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	err = z.Sequencer.DecodeMsg(dc)
	if err != nil {
		err = msgp.WrapError(err, "Sequencer")
		return
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *SequencerProposal) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = z.Sequencer.EncodeMsg(en)
	if err != nil {
		err = msgp.WrapError(err, "Sequencer")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *SequencerProposal) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o, err = z.Sequencer.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Sequencer")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *SequencerProposal) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	bts, err = z.Sequencer.UnmarshalMsg(bts)
	if err != nil {
		err = msgp.WrapError(err, "Sequencer")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *SequencerProposal) Msgsize() (s int) {
	s = 1 + z.Sequencer.Msgsize()
	return
}

