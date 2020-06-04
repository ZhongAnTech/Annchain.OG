package og

//go:generate msgp

//msgp BouncerMessage
type BouncerMessage struct {
	Value int
}
