package types

import (
	"errors"
)

var (
	ErrDuplicateTx = errors.New("Duplicate tx found in txlookup")

	ErrDuplicateNonce = errors.New("Duplicate tx nonce")
)
