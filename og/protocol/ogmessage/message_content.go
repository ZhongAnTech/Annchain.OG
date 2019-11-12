package ogmessage

type ResourceType uint8

//go:generate msgp

//msgp:tuple MessageContentResource
type MessageContentResource struct {
	ResourceType    ResourceType
	ResourceContent []byte
}
