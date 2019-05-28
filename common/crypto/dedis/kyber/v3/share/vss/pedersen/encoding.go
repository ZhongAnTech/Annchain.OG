package vss

import "fmt"

func (d *EncryptedDeal) String() string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("encDeal-%x", d.Cipher[:])
}

func (d *EncryptedDeal) TerminateString() string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("encdeal-len-%d", len(d.Cipher))
}
