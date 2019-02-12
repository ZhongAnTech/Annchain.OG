package cosi

import "github.com/annchain/OG/common/crypto/dedis/kyber/v3"

// Suite specifies the cryptographic building blocks required for the cosi package.
type Suite interface {
	kyber.Group
	kyber.HashFactory
	kyber.Random
}
