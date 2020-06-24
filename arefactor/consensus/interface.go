package consensus

type Signable interface {
	SignatureTarget() []byte
}
