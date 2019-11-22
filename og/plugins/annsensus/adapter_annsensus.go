package annsensus

//type AnnsensusMessageUnmarshaller struct {
//}
//
//func (a AnnsensusMessageUnmarshaller) Unmarshal(
//	messageType annsensus.AnnsensusMessageType, message []byte) (
//	outMsg annsensus.AnnsensusMessage, err error) {
//
//	return
//}
//
//// SimpleAnnsensusAdapter adapts a transportable message to an annsensus message
//// requires the message type to be MessageTypeAnnsensus
//type SimpleAnnsensusAdapter struct {
//	annsensusMessageUnmarshaller *AnnsensusMessageUnmarshaller
//}
//
//func (s SimpleAnnsensusAdapter) AdaptMessage(incomingMsg msg.OgMessage) (annMsg annsensus.AnnsensusMessage, err error) {
//	mssageType := og.OgMessageType(incomingMsg.GetType())
//	if mssageType != og.MessageTypeAnnsensus {
//		err = errors.New("SimpleAnnsensusAdapter received a message of an unsupported type")
//		return
//	}
//	// incomingMsg.GetType() == types.MessageTypeAnnsensus
//	// incomingMsg.GetData
//	return s.annsensusMessageUnmarshaller.Unmarshal(annsensus.AnnsensusMessageType(incomingMsg.GetType()), incomingMsg.GetData())
//}
//
//func (s SimpleAnnsensusAdapter) AdaptAnnsensusMessage(outgoingMsg annsensus.AnnsensusMessage) (msg.OgMessage, error) {
//	panic("implement me")
//}
