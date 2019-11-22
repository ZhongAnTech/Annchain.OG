package annsensus

type AnnsensusPartner struct {
}

func (a AnnsensusPartner) Broadcast(msg AnnsensusMessage, peers []AnnsensusPeer) {
	adaptedMessage, err := ap.bftMessageAdapter.AdaptBftMessage(msg)

}

func (a AnnsensusPartner) Unicast(msg AnnsensusMessage, peer AnnsensusPeer) {
	panic("implement me")
}

func (a AnnsensusPartner) GetPipeIn() chan AnnsensusMessage {
	panic("implement me")
}

func (a AnnsensusPartner) GetPipeOut() chan AnnsensusMessage {
	panic("implement me")
}
