package message

type ResourceType uint8

const (
	ResourceTypeTx ResourceType = iota
	ResourceTypeSequencer
	ResourceTypeArchive
	ResourceTypeAction
)

//go:generate msgp

//msgp:tuple MessageContentResource
type MessageContentResource struct {
	ResourceType    ResourceType
	ResourceContent []byte
}
