package ogsyncer_interface

type Resource interface {
	GetType() int
	ToBytes() []byte
	FromBytes([]byte) error
}

type ResourceRequest interface {
	GetType() int
	ToBytes() []byte
	FromBytes([]byte) error
}

type ResourceFetcher interface {
	Fetch(request ResourceRequest) []Resource
}
