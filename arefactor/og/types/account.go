package types

import (
	"github.com/annchain/OG/arefactor/og_interface"
	"github.com/annchain/OG/arefactor/ogcrypto_interface"
)

// OgAccount represents a full account of a user.
type OgAccount struct {
	PublicKey  ogcrypto_interface.PublicKey
	PrivateKey ogcrypto_interface.PrivateKey
	Address    og_interface.Address
}
