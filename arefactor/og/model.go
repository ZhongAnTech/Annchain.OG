package og

//go:generate msgp

//msgp:tuple BouncerMessage
type BouncerMessage struct {
	Value int
}
