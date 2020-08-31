package core

type Component interface {
	Start()
	Stop()
	// Get the component name
	Name() string
}

type EventInfoProvider interface {
	GetEventMap() map[int]string
}
