package dkg

import "fmt"

func (d Deal) String() string {
	return fmt.Sprintf("deal-%d-%s", d.Index, d.Deal)
}

func (d Deal) TerminateString() string {
	return fmt.Sprintf("deal-%d-%s", d.Index, d.Deal.TerminateString())
}
