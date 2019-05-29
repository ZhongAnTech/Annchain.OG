package pairing

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/util/key"
)

func TestAdapter_SuiteBn256(t *testing.T) {
	suite := NewSuiteBn256()

	pair := key.NewKeyPair(suite)
	pubkey, err := pair.Public.MarshalBinary()
	require.Nil(t, err)
	privkey, err := pair.Private.MarshalBinary()
	require.Nil(t, err)

	pubhex := suite.Point()
	err = pubhex.UnmarshalBinary(pubkey)
	require.Nil(t, err)

	privhex := suite.Scalar()
	err = privhex.UnmarshalBinary(privkey)
	require.Nil(t, err)

	require.Equal(t, "bn256.adapter", suite.String())
}
