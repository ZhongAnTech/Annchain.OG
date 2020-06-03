package core

type Component interface {
	Start()
	Stop()
	// Get the component name
	Name() string
}

type AccountHolder interface {
	ProvidePrivateKey(createIfMissing bool) ([]byte, error)
}
