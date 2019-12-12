package message_archive

//type MsgKey struct {
//	data [common.HashLength + 2]byte
//}
//
//func (k MsgKey) GetType() (BinaryMessageType, error) {
//	if len(k.data) != common.HashLength+2 {
//		return 0, errors.New("size err")
//	}
//	return BinaryMessageType(binary.BigEndian.Uint16(k.data[0:2])), nil
//}
//
//func (k MsgKey) GetHash() (common.Hash, error) {
//	if len(k.data) != common.HashLength+2 {
//		return common.Hash{}, errors.New("size err")
//	}
//	return common.BytesToHash(k.data[2:]), nil
//}
//
//func NewMsgKey(m BinaryMessageType, hash common.Hash) MsgKey {
//	var key MsgKey
//	b := make([]byte, 2)
//	//use one key for tx and sequencer
//	if m == MessageTypeNewSequencer {
//		m = MessageTypeNewTx
//	}
//	binary.BigEndian.PutUint16(b, uint16(m))
//	copy(key.data[:], b)
//	copy(key.data[2:], hash.ToBytes())
//	return key
//}
