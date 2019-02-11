// Package anon implements cryptographic primitives for anonymous communication.
package anon

import (
	"github.com/annchain/OG/common/crypto/dedis/kyber/v3"
)

// Set represents an explicit anonymity set
// as a list of public keys.
type Set []kyber.Point
