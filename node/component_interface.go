package node

type Component interface {
	Start()
	Stop()
	// Get the component name
	Name() string
}
