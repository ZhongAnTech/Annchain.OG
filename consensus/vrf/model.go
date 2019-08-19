package vrf

import (
	"encoding/hex"
	"fmt"
)

//msgp:tuple VrfInfo
type VrfInfo struct {
	Message   []byte
	Proof     []byte
	PublicKey []byte
	Vrf       []byte
}


func (v *VrfInfo) String() string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("Msg:%s, vrf :%s , proof :%s, pubKey :%s", hex.EncodeToString(v.Message), hex.EncodeToString(v.Vrf),
		hex.EncodeToString(v.Proof), hex.EncodeToString(v.PublicKey))
}
